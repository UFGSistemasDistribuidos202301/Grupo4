package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"

	bolt "go.etcd.io/bbolt"
)

var (
	DB *bolt.DB
)

func getDBDir() string {
	return fmt.Sprintf("./node_%d", *nodeID)
}

func openDB() {
	var err error
	DB, err = bolt.Open(getDBDir(), 0600, nil)
	if err != nil {
		log.Fatal(err)
	}
}

func writeError(w http.ResponseWriter, msg string) {
	type Error struct {
		Message string `json:"message"`
	}

	errObj := Error{Message: msg}
	errBytes, err := json.Marshal(errObj)
	if err != nil {
		log.Panic(err)
	}

	w.WriteHeader(500)
	w.Write(errBytes)
}

func writeBodyJson(w http.ResponseWriter, bodyJson any) {
	bodyBytes, err := json.Marshal(bodyJson)
	if err != nil {
		log.Panic(err)
	}
	_, err = w.Write(bodyBytes)
	if err != nil {
		log.Panic(err)
	}
}

func getBodyMap(r *http.Request) map[string]string {
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Panic(err)
	}

	var bodyMap map[string]string
	err = json.Unmarshal(body, &bodyMap)
	if err != nil {
		log.Panic(err)
	}

	return bodyMap
}

func getItem(txn *bolt.Tx, tableName string, docId string) map[string]string {
	bucket := txn.Bucket([]byte(tableName))
	if bucket == nil {
		return nil
	}
	item := bucket.Get([]byte(docId))
	if item == nil {
		return nil
	}

	var itemMap map[string]string
	err := json.Unmarshal(item, &itemMap)
	if err != nil {
		log.Panic(err)
	}

	return itemMap
}

func setItem(
	txn *bolt.Tx,
	tableName string,
	docId string,
	itemMap map[string]string,
) error {
	bucket := txn.Bucket([]byte(tableName))
	if bucket == nil {
		return errors.New("table does not exist")
	}

	value, err := json.Marshal(itemMap)
	if err != nil {
		return err
	}
	err = bucket.Put([]byte(docId), value)
	if err != nil {
		return err
	}
	return nil
}
