package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
	"sync"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/gorilla/websocket"

	bolt "go.etcd.io/bbolt"
)

type VisEvent struct {
	NodeID uint   `json:"node_id"`
	Kind   string `json:"kind"`
	Data   any    `json:"data"`
}

var (
	upgrader = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool { return true },
	}
	wsListeners      = make(map[*websocket.Conn]chan<- VisEvent)
	wsListenersLock  = sync.Mutex{}
	visEventsChannel = make(chan VisEvent)
)

type TableCreationParams struct {
	StrongConsistency bool `json:"strong_consistency"`
}

func fileServer(r chi.Router, path string, root http.FileSystem) {
	if strings.ContainsAny(path, "{}*") {
		panic("FileServer does not permit any URL parameters.")
	}

	if path != "/" && path[len(path)-1] != '/' {
		r.Get(path, http.RedirectHandler(path+"/", 301).ServeHTTP)
		path += "/"
	}
	path += "*"

	r.Get(path, func(w http.ResponseWriter, r *http.Request) {
		rctx := chi.RouteContext(r.Context())
		pathPrefix := strings.TrimSuffix(rctx.RoutePattern(), "/*")
		fs := http.StripPrefix(pathPrefix, http.FileServer(root))
		fs.ServeHTTP(w, r)
	})
}

func startHTTPServer() {
	go func() {
		for msg := range visEventsChannel {
			wsListenersLock.Lock()
			for _, recv := range wsListeners {
				recv <- msg
			}
			wsListenersLock.Unlock()
		}
	}()

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	// Websocket endpoint
	r.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		// Upgrade connection to websocket protocol and send events to client
		conn, err := upgrader.Upgrade(w, r, nil)

		if err != nil {
			log.Printf("error: %v\n", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		defer conn.Close()

		recv := make(chan VisEvent)
		wsListenersLock.Lock()
		wsListeners[conn] = recv
		wsListenersLock.Unlock()
		defer func() {
			wsListenersLock.Lock()
			defer wsListenersLock.Unlock()
			delete(wsListeners, conn)
		}()

		// Send all events to client until connection is closed
		for event := range recv {
			eventJson, err := json.Marshal(event)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			if err := conn.WriteMessage(1, eventJson); err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
		}
	})

	filesDir := http.Dir(http.Dir("visualization"))
	fileServer(r, "/visualization", filesDir)

	// PUT /<table> (body: { "strong_consistency": "true" })
	// Cria uma tabela (indicando se será eventual ou forte)
	r.Put("/db/{tableName}", func(w http.ResponseWriter, r *http.Request) {
		tableName := chi.URLParam(r, "tableName")

		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			log.Panic(err)
		}

		var params TableCreationParams
		err = json.Unmarshal(body, &params)
		if err != nil {
			log.Panic(err)
		}

		err = DB.OpenTx(func(tx *bolt.Tx) error {
			_, err := DB.CreateTable(tx, tableName, params.StrongConsistency)
			if err != nil {
				return err
			}
			return nil
		})
		if err != nil {
			writeError(w, "could not create table: "+err.Error())
			return
		}

		// Send event
		visEventsChannel <- VisEvent{
			NodeID: *nodeID,
			Kind:   "create_table",
			Data:   tableName,
		}
	})

	// DELETE /<table>
	// Remove uma tabela
	r.Delete("/db/{tableName}", func(w http.ResponseWriter, r *http.Request) {
		tableName := chi.URLParam(r, "tableName")

		err := DB.OpenTx(func(tx *bolt.Tx) error {
			return DB.DeleteTable(tx, tableName)
		})
		if err != nil {
			writeError(w, "could not create table: "+err.Error())
			return
		}

		// Send event
		visEventsChannel <- VisEvent{
			NodeID: *nodeID,
			Kind:   "delete_table",
			Data:   tableName,
		}
	})

	// GET /<table>/<doc id>
	// Retorna o mapa de chaves e valores de um documento
	r.Get("/db/{tableName}/{docId}", func(w http.ResponseWriter, r *http.Request) {
		tableName := chi.URLParam(r, "tableName")
		docId := chi.URLParam(r, "docId")

		err := DB.OpenTx(func(tx *bolt.Tx) error {
			table, err := DB.GetTable(tx, tableName)
			if err != nil {
				return err
			}
			doc, err := table.Get(tx, docId)
			if err != nil {
				return err
			}

			writeBodyJson(w, doc)
			return nil
		})

		if err != nil {
			writeError(w, "document does not exist")
			return
		}
	})

	// PUT /<table>/<doc id> (body: keys/values)
	// Substitui o documento inteiro, apagando outras
	// chaves não especificadas nesta chamada
	r.Put("/db/{tableName}/{docId}", func(w http.ResponseWriter, r *http.Request) {
		tableName := chi.URLParam(r, "tableName")
		docId := chi.URLParam(r, "docId")
		bodyMap := getBodyMap(r)

		err := DB.OpenTx(func(tx *bolt.Tx) error {
			table, err := DB.GetTable(tx, tableName)
			if err != nil {
				return err
			}

			err = table.Put(tx, docId, bodyMap)
			if err != nil {
				return err
			}

			writeBodyJson(w, bodyMap)

			// Send event
			visEventsChannel <- VisEvent{
				NodeID: *nodeID,
				Kind:   "put_document",
				Data:   map[string]any{docId: bodyMap},
			}

			return nil
		})

		if err != nil {
			writeError(w, "error putting document: "+err.Error())
			return
		}
	})

	// PATCH /<table>/<doc id> (body: keys/values)
	// Substitui valores dentro de um documento,
	// mantendo chaves já existentes que não foram especificadas nesta chamada
	// Antes: {"chave1": "valor1", "chave2": "valor2"}
	// PATCH {"chave3": "valor3", "chave2": "asdasdasdas"}
	// Depois: {"chave1": "valor1", "chave2": "asdasdasdas", "chave3": "valor3"}
	r.Patch("/db/{tableName}/{docId}", func(w http.ResponseWriter, r *http.Request) {
		tableName := chi.URLParam(r, "tableName")
		docId := chi.URLParam(r, "docId")
		bodyMap := getBodyMap(r)

		err := DB.OpenTx(func(tx *bolt.Tx) error {
			table, err := DB.GetTable(tx, tableName)
			if err != nil {
				return err
			}

			newDoc, err := table.Patch(tx, docId, bodyMap)
			if err != nil {
				return err
			}

			writeBodyJson(w, newDoc)

			// Send event
			visEventsChannel <- VisEvent{
				NodeID: *nodeID,
				Kind:   "patch_document",
				Data:   map[string]any{docId: newDoc},
			}

			return nil
		})

		if err != nil {
			writeError(w, "error patching document: "+err.Error())
			return
		}
	})

	// DELETE /<table>/<doc id>
	// Remove um documento
	r.Delete("/db/{tableName}/{docId}", func(w http.ResponseWriter, r *http.Request) {
		tableName := chi.URLParam(r, "tableName")
		docId := chi.URLParam(r, "docId")

		err := DB.OpenTx(func(tx *bolt.Tx) error {
			table, err := DB.GetTable(tx, tableName)
			if err != nil {
				return err
			}
			deletedDoc, err := table.Delete(tx, docId)
			if err != nil {
				return err
			}

			writeBodyJson(w, deletedDoc)
			return nil
		})
		if err != nil {
			writeError(w, "document does not exist")
			return
		}

		// Send event
		visEventsChannel <- VisEvent{
			NodeID: *nodeID,
			Kind:   "delete_document",
			Data:   docId,
		}
	})

	// GET /<table>
	// Retorna todos os documentos de uma tabela
	r.Get("/db/{tableName}", func(w http.ResponseWriter, r *http.Request) {
		tableName := chi.URLParam(r, "tableName")

		tableValues := map[string]map[string]string{}

		err := DB.OpenTx(func(tx *bolt.Tx) error {
			table, err := DB.GetTable(tx, tableName)
			if err != nil {
				return err
			}

			return table.ForEach(tx, func(docId string, doc map[string]string) error {
				tableValues[docId] = doc
				return nil
			})
		})
		if err != nil {
			writeError(w, "failed to get table documents: "+err.Error())
			return
		}

		writeBodyJson(w, tableValues)

		// Send event
		visEventsChannel <- VisEvent{
			NodeID: *nodeID,
			Kind:   "get_table_documents",
			Data:   map[string]any{tableName: tableValues},
		}
	})

	// GET /
	// Retorna todos os documentos de todas as tabelas
	r.Get("/db", func(w http.ResponseWriter, r *http.Request) {
		dbValues := map[string]map[string]map[string]string{}

		err := DB.OpenTx(func(tx *bolt.Tx) error {
			return DB.ForEach(tx, func(tableName string, table Table) error {
				if _, ok := dbValues[tableName]; !ok {
					dbValues[tableName] = map[string]map[string]string{}
				}
				return table.ForEach(tx, func(docId string, doc map[string]string) error {
					dbValues[tableName][docId] = doc
					return nil
				})
			})
		})
		if err != nil {
			writeError(w, "failed to get table documents: "+err.Error())
			return
		}

		writeBodyJson(w, dbValues)

		// Send event
		visEventsChannel <- VisEvent{
			NodeID: *nodeID,
			Kind:   "get_all_docs",
			Data:   dbValues,
		}

	})

	addr := fmt.Sprintf(":%d", httpPort)
	log.Printf("HTTP server listening at %s\n", addr)
	http.ListenAndServe(addr, r)
}
