package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"

	"github.com/dgraph-io/badger/v4"
)

var (
	tables  map[string]*badger.DB = make(map[string]*badger.DB)
)

func getRootDir() string {
	return fmt.Sprintf("./node_%d", *nodeID)
}

func loadTables() error {
	files, err := ioutil.ReadDir(getRootDir())
	if err != nil {
		return err
	}

	for _, file := range files {
		tableName := file.Name()
		db, err := badger.Open(badger.DefaultOptions(getRootDir() + "/" + tableName))
		if err != nil {
			return err
		}
		tables[tableName] = db
	}

	return nil
}

func getTable(name string, create bool) (*badger.DB, error) {
	dbPath := fmt.Sprintf("%s/%s", getRootDir(), name)
	_, err := os.Stat(dbPath)
	dirExists := err == nil
	if !create && !dirExists {
		return nil, errors.New("table does not exist")
	}
	db, exists := tables[name]
	if exists {
		return db, nil
	}

	db, err = badger.Open(badger.DefaultOptions(dbPath))
	if err != nil {
		return nil, err
	}
	tables[name] = db
	return db, nil
}

func writeError(w http.ResponseWriter, msg string) {
	type Error struct {
		Message string `json:"message"`
	}

	errObj := Error{Message: msg}
	errBytes, err := json.Marshal(errObj)
	if err != nil {
		panic(err)
	}

	w.WriteHeader(500)
	w.Write(errBytes)
}

func writeBodyJson(w http.ResponseWriter, bodyJson any) error {
	bodyBytes, err := json.Marshal(bodyJson)
	if err != nil {
		return err
	}
	_, err = w.Write(bodyBytes)
	if err != nil {
		return err
	}

	return nil
}

func getBodyMap(r *http.Request) (map[string]string, error) {
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return nil, errors.New("error reading request body")
	}

	var bodyMap map[string]string
	err = json.Unmarshal(body, &bodyMap)
	if err != nil {
		return nil, errors.New("error parsing request body")
	}

	return bodyMap, nil
}

func getItem(txn *badger.Txn, docId string) (map[string]string, error) {
	item, err := txn.Get([]byte(docId))
	if err != nil {
		return nil, err
	}

	itemBytes, err := item.ValueCopy(nil)
	if err != nil {
		return nil, err
	}

	var itemMap map[string]string
	err = json.Unmarshal(itemBytes, &itemMap)
	if err != nil {
		return nil, errors.New("error parsing item json")
	}

	return itemMap, nil
}

func setItem(
	txn *badger.Txn,
	docId string,
	itemMap map[string]string,
) error {
	value, err := json.Marshal(itemMap)
	if err != nil {
		return err
	}
	err = txn.Set([]byte(docId), value)
	if err != nil {
		return err
	}
	return nil
}
