package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
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

func (i *Instance) startHTTPServer() {
	go func() {
		for msg := range i.visEventsChannel {
			i.wsListenersLock.Lock()
			for _, recv := range i.wsListeners {
				recv <- msg
			}
			i.wsListenersLock.Unlock()
		}
	}()

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(cors.Handler(cors.Options{
		AllowOriginFunc:  func(r *http.Request, origin string) bool { return true },
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: false,
		MaxAge:           300, // Maximum value not ignored by any of major browsers
	}))

	// Websocket endpoint
	r.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		// Upgrade connection to websocket protocol and send events to client
		conn, err := upgrader.Upgrade(w, r, nil)

		if err != nil {
			i.logger.Printf("error: %v\n", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		defer conn.Close()

		recv := make(chan VisEvent)
		i.wsListenersLock.Lock()
		i.wsListeners[conn] = recv
		i.wsListenersLock.Unlock()
		defer func() {
			i.wsListenersLock.Lock()
			defer i.wsListenersLock.Unlock()
			delete(i.wsListeners, conn)
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
		if i.isOffline() {
			log.Panic("Node is offline")
		}

		tableName := chi.URLParam(r, "tableName")

		body, err := io.ReadAll(r.Body)
		if err != nil {
			log.Panic(err)
		}

		var params TableCreationParams
		err = json.Unmarshal(body, &params)
		if err != nil {
			log.Panic(err)
		}

		err = i.db.OpenTx(func(tx *bolt.Tx) error {
			_, err := i.db.CreateTable(tx, tableName, params.StrongConsistency)
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
		i.visEventsChannel <- VisEvent{
			NodeID: i.nodeID,
			Kind:   "create_table",
			Data:   tableName,
		}
	})

	// DELETE /<table>
	// Remove uma tabela
	r.Delete("/db/{tableName}", func(w http.ResponseWriter, r *http.Request) {
		if i.isOffline() {
			log.Panic("Node is offline")
		}

		tableName := chi.URLParam(r, "tableName")

		err := i.db.OpenTx(func(tx *bolt.Tx) error {
			return i.db.DeleteTable(tx, tableName)
		})
		if err != nil {
			writeError(w, "could not create table: "+err.Error())
			return
		}

		// Send event
		i.visEventsChannel <- VisEvent{
			NodeID: i.nodeID,
			Kind:   "delete_table",
			Data:   tableName,
		}
	})

	// GET /<table>/<doc id>
	// Retorna o mapa de chaves e valores de um documento
	r.Get("/db/{tableName}/{docId}", func(w http.ResponseWriter, r *http.Request) {
		if i.isOffline() {
			log.Panic("Node is offline")
		}

		tableName := chi.URLParam(r, "tableName")
		docId := chi.URLParam(r, "docId")

		err := i.db.OpenTx(func(tx *bolt.Tx) error {
			table, err := i.db.GetTable(tx, tableName)
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
		if i.isOffline() {
			log.Panic("Node is offline")
		}

		tableName := chi.URLParam(r, "tableName")
		docId := chi.URLParam(r, "docId")
		bodyMap := getBodyMap(r)

		err := i.db.OpenTx(func(tx *bolt.Tx) error {
			table, err := i.db.GetTable(tx, tableName)
			if err != nil {
				return err
			}

			err = table.Put(tx, docId, bodyMap)
			if err != nil {
				return err
			}

			writeBodyJson(w, bodyMap)

			// Send event
			i.visEventsChannel <- VisEvent{
				NodeID: i.nodeID,
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
		if i.isOffline() {
			log.Panic("Node is offline")
		}

		tableName := chi.URLParam(r, "tableName")
		docId := chi.URLParam(r, "docId")
		bodyMap := getBodyMap(r)

		err := i.db.OpenTx(func(tx *bolt.Tx) error {
			table, err := i.db.GetTable(tx, tableName)
			if err != nil {
				return err
			}

			newDoc, err := table.Patch(tx, docId, bodyMap)
			if err != nil {
				return err
			}

			writeBodyJson(w, newDoc)

			// Send event
			i.visEventsChannel <- VisEvent{
				NodeID: i.nodeID,
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
		if i.isOffline() {
			log.Panic("Node is offline")
		}

		tableName := chi.URLParam(r, "tableName")
		docId := chi.URLParam(r, "docId")

		err := i.db.OpenTx(func(tx *bolt.Tx) error {
			table, err := i.db.GetTable(tx, tableName)
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
		i.visEventsChannel <- VisEvent{
			NodeID: i.nodeID,
			Kind:   "delete_document",
			Data:   docId,
		}
	})

	// GET /<table>
	// Retorna todos os documentos de uma tabela
	r.Get("/db/{tableName}", func(w http.ResponseWriter, r *http.Request) {
		if i.isOffline() {
			log.Panic("Node is offline")
		}

		tableName := chi.URLParam(r, "tableName")

		tableValues := map[string]map[string]string{}

		err := i.db.OpenTx(func(tx *bolt.Tx) error {
			table, err := i.db.GetTable(tx, tableName)
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
	})

	// GET /
	// Retorna todos os documentos de todas as tabelas
	r.Get("/db", func(w http.ResponseWriter, r *http.Request) {
		if i.isOffline() {
			log.Panic("Node is offline")
		}

		dbValues := map[string]map[string]map[string]string{}

		err := i.db.OpenTx(func(tx *bolt.Tx) error {
			return i.db.ForEach(tx, func(tableName string, table Table) error {
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
	})

	r.Put("/crdt_sync", func(w http.ResponseWriter, r *http.Request) {
		if i.isOffline() {
			log.Panic("Node is offline")
		}

		i.syncPendingCRDTStates()
	})

	r.Put("/offline", func(w http.ResponseWriter, r *http.Request) {
		i.offlineMutex.Lock()
		i.offline = true
		i.offlineMutex.Unlock()
	})

	r.Put("/online", func(w http.ResponseWriter, r *http.Request) {
		i.offlineMutex.Lock()
		i.offline = false
		i.offlineMutex.Unlock()
	})

	addr := fmt.Sprintf(":%d", i.httpPort)
	i.logger.Printf("HTTP server listening at %s\n", addr)
	http.ListenAndServe(addr, r)
}
