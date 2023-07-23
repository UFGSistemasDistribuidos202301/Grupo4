package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"

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
	upgrader         = websocket.Upgrader{} // use default options
	visEventsChannel = make(chan VisEvent, 2048)

	nodeID       = flag.Uint("id", 0, "The ID of this node")
	baseHTTPPort = flag.Uint("baseHTTPPort", 3000, "The base HTTP port")
)

func main() {
	flag.Parse()
	*baseHTTPPort += *nodeID

	openDB()

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	// PUT /<table> (body: { "strong_consistency": "true" })
	// Cria uma tabela (indicando se será eventual ou forte)
	r.Put("/{tableName}", func(w http.ResponseWriter, r *http.Request) {
		tableName := chi.URLParam(r, "tableName")

		err := DB.Update(func(tx *bolt.Tx) error {
			existingBucket := tx.Bucket([]byte(tableName))
			if existingBucket != nil {
				return errors.New("table already exists")
			}

			if _, err := tx.CreateBucket([]byte(tableName)); err != nil {
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
	r.Delete("/{tableName}", func(w http.ResponseWriter, r *http.Request) {
		tableName := chi.URLParam(r, "tableName")

		err := DB.Update(func(tx *bolt.Tx) error {
			existingBucket := tx.Bucket([]byte(tableName))
			if existingBucket == nil {
				return errors.New("table does not exist")
			}

			err := tx.DeleteBucket([]byte(tableName))
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
			Kind:   "delete_table",
			Data:   tableName,
		}
	})

	// GET /<table>/<doc id>
	// Retorna o mapa de chaves e valores de um documento
	r.Get("/{tableName}/{docId}", func(w http.ResponseWriter, r *http.Request) {
		tableName := chi.URLParam(r, "tableName")
		docId := chi.URLParam(r, "docId")

		err := DB.View(func(txn *bolt.Tx) error {
			itemMap := getItem(txn, tableName, docId)
			if itemMap == nil {
				return errors.New("document does not exist")
			}
			writeBodyJson(w, itemMap)

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

		err := DB.Update(func(txn *bolt.Tx) error {
			bodyMap := getBodyMap(r)
			if err := setItem(txn, tableName, docId, bodyMap); err != nil {
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
	r.Patch("/{tableName}/{docId}", func(w http.ResponseWriter, r *http.Request) {
		tableName := chi.URLParam(r, "tableName")
		docId := chi.URLParam(r, "docId")

		err := DB.Update(func(txn *bolt.Tx) error {
			itemMap := getItem(txn, tableName, docId)
			if itemMap == nil {
				itemMap = map[string]string{}
			}

			bodyMap := getBodyMap(r)

			for k, v := range bodyMap {
				itemMap[k] = v
			}

			if err := setItem(txn, tableName, docId, itemMap); err != nil {
				return err
			}

			writeBodyJson(w, itemMap)

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

		err := DB.Update(func(txn *bolt.Tx) error {
			bucket := txn.Bucket([]byte(tableName))
			if bucket == nil {
				return errors.New("table does not exist")
			}

			itemMap := getItem(txn, tableName, docId)
			if itemMap == nil {
				return errors.New("document does not exist")
			}

			if err := bucket.Delete([]byte(docId)); err != nil {
				return err
			}

			writeBodyJson(w, itemMap)

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

		tableValues := map[string]map[string]string{}

		err := DB.View(func(tx *bolt.Tx) error {
			bucket := tx.Bucket([]byte(tableName))
			if bucket == nil {
				return errors.New("table does not exist")
			}
			return bucket.ForEach(func(key, value []byte) error {
				docId := string(key)
				docValue := getItem(tx, tableName, docId)
				tableValues[docId] = docValue
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
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		dbValues := map[string]map[string]map[string]string{}

		err := DB.View(func(tx *bolt.Tx) error {
			return tx.ForEach(func(name []byte, bucket *bolt.Bucket) error {
				tableName := string(name)
				dbValues[tableName] = map[string]map[string]string{}

				return bucket.ForEach(func(key, value []byte) error {
					docId := string(key)
					docValue := getItem(tx, tableName, docId)
					dbValues[tableName][docId] = docValue
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

	log.SetPrefix(fmt.Sprintf("[NODE #%d] ", *nodeID))
	log.Printf("Launching node #%d", *nodeID)
	http.ListenAndServe(fmt.Sprintf(":%d", *baseHTTPPort), r)
}
