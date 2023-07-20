package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"os"

	"github.com/dgraph-io/badger/v4"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/gorilla/websocket"
)

type VisEvent struct {
	NodeID uint   `json:"node_id"`
	Kind   string `json:"kind"`
	Data   any    `json:"data"`
}

var (
	upgrader         = websocket.Upgrader{} // use default options
	visEventsChannel = make(chan VisEvent)

	nodeID       = flag.Uint("id", 0, "The ID of this node")
	baseHTTPPort = flag.Uint("baseHTTPPort", 3000, "The base HTTP port")
)

func main() {
	flag.Parse()
	*baseHTTPPort += *nodeID

	if err := loadTables(); err != nil {
		panic(err)
	}

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	// PUT /<table> (body: { "strong_consistency": "true" })
	// Cria uma tabela (indicando se será eventual ou forte)
	r.Put("/{tableName}", func(w http.ResponseWriter, r *http.Request) {
		tableName := chi.URLParam(r, "tableName")
		_, err := getTable(tableName, false)
		if err == nil {
			writeError(w, "table already exists")
			return
		}

		_, err = getTable(tableName, true)
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
	r.Delete("/{tableName}", func(w http.ResponseWriter, r *http.Request) {
		tableName := chi.URLParam(r, "tableName")
		table, err := getTable(tableName, false)
		if err != nil {
			writeError(w, "table does not exist")
			return
		}

		table.Close()
		delete(tables, tableName)

		// Send event
		visEventsChannel <- VisEvent{
			NodeID: *nodeID,
			Kind:   "delete_table",
			Data:   tableName,
		}

		tablePath := fmt.Sprintf("%s/%s", getRootDir(), tableName)
		err = os.RemoveAll(tablePath)
		if err != nil {
			writeError(w, "error deleting table")
			return
		}
	})

	// GET /<table>/<doc id>
	// Retorna o mapa de chaves e valores de um documento
	r.Get("/{tableName}/{docId}", func(w http.ResponseWriter, r *http.Request) {
		tableName := chi.URLParam(r, "tableName")
		docId := chi.URLParam(r, "docId")
		table, err := getTable(tableName, false)
		if err != nil {
			writeError(w, "table does not exist")
			return
		}

		err = table.View(func(txn *badger.Txn) error {
			itemMap, err := getItem(txn, docId)
			if err != nil {
				return err
			}
			err = writeBodyJson(w, itemMap)
			if err != nil {
				return err
			}

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
	r.Put("/{tableName}/{docId}", func(w http.ResponseWriter, r *http.Request) {
		tableName := chi.URLParam(r, "tableName")
		docId := chi.URLParam(r, "docId")
		table, err := getTable(tableName, false)
		if err != nil {
			writeError(w, "table does not exist")
			return
		}

		err = table.Update(func(txn *badger.Txn) error {
			bodyMap, err := getBodyMap(r)
			if err != nil {
				return err
			}

			err = setItem(txn, docId, bodyMap)
			if err != nil {
				return err
			}

			err = writeBodyJson(w, bodyMap)
			if err != nil {
				return err
			}

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
	r.Patch("/{tableName}/{docId}", func(w http.ResponseWriter, r *http.Request) {
		tableName := chi.URLParam(r, "tableName")
		docId := chi.URLParam(r, "docId")
		table, err := getTable(tableName, false)
		if err != nil {
			writeError(w, "table does not exist")
			return
		}

		err = table.Update(func(txn *badger.Txn) error {
			itemMap, err := getItem(txn, docId)
			if err != nil {
				itemMap = map[string]string{}
			}

			bodyMap, err := getBodyMap(r)
			if err != nil {
				return err
			}

			for k, v := range bodyMap {
				itemMap[k] = v
			}

			err = setItem(txn, docId, itemMap)
			if err != nil {
				return err
			}

			err = writeBodyJson(w, itemMap)
			if err != nil {
				return err
			}

			// Send event
			visEventsChannel <- VisEvent{
				NodeID: *nodeID,
				Kind:   "patch_document",
				Data:   map[string]any{docId: itemMap},
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
	r.Delete("/{tableName}/{docId}", func(w http.ResponseWriter, r *http.Request) {
		tableName := chi.URLParam(r, "tableName")
		docId := chi.URLParam(r, "docId")
		table, err := getTable(tableName, false)
		if err != nil {
			writeError(w, "table does not exist")
			return
		}

		err = table.Update(func(txn *badger.Txn) error {
			itemMap, err := getItem(txn, docId)
			if err != nil {
				return errors.New("document does not exist")
			}

			err = txn.Delete([]byte(docId))
			if err != nil {
				return err
			}

			err = writeBodyJson(w, itemMap)
			if err != nil {
				return err
			}

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
	r.Get("/{tableName}", func(w http.ResponseWriter, r *http.Request) {
		tableName := chi.URLParam(r, "tableName")
		table, err := getTable(tableName, false)
		if err != nil {
			writeError(w, "table does not exist")
			return
		}

		err = table.View(func(txn *badger.Txn) error {
			it := txn.NewIterator(badger.DefaultIteratorOptions)
			defer it.Close()

			tableValues := map[string]map[string]string{}

			for it.Rewind(); it.Valid(); it.Next() {
				docId := string(it.Item().Key())
				docValue, err := getItem(txn, docId)
				if err != nil {
					return err
				}
				tableValues[docId] = docValue
			}

			err = writeBodyJson(w, tableValues)
			if err != nil {
				return err
			}

			return nil
		})

		if err != nil {
			writeError(w, "failed to get table documents: "+err.Error())
			return
		}
	})

	// GET /
	// Retorna todos os documentos de todas as tabelas
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		dbValues := map[string]map[string]map[string]string{}

		for tableName, table := range tables {
			dbValues[tableName] = map[string]map[string]string{}
			err := table.View(func(txn *badger.Txn) error {
				it := txn.NewIterator(badger.DefaultIteratorOptions)
				defer it.Close()

				for it.Rewind(); it.Valid(); it.Next() {
					docId := string(it.Item().Key())
					docValue, err := getItem(txn, docId)
					if err != nil {
						return err
					}
					dbValues[tableName][docId] = docValue
				}

				return nil
			})

			if err != nil {
				writeError(w, "failed to get table documents: "+err.Error())
				return
			}
		}

		err := writeBodyJson(w, dbValues)
		if err != nil {
			writeError(w, "failed to get DB documents: "+err.Error())
		}
	})

	// Websocket endpoint
	r.Get("/ws", func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		defer conn.Close()
		for {
			event := <-visEventsChannel
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

	http.ListenAndServe(fmt.Sprintf(":%d", *baseHTTPPort), r)
}
