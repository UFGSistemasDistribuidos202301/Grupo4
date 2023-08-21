package main

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"log"
	"net/http"

	bolt "go.etcd.io/bbolt"
)

type Table interface {
	Put(tx *bolt.Tx, docId string, doc map[string]string) error
	Patch(tx *bolt.Tx, docId string, doc map[string]string) (map[string]string, error)
	Delete(tx *bolt.Tx, docId string) (map[string]string, error)
	Get(tx *bolt.Tx, docId string) (map[string]string, error)
	ForEach(tx *bolt.Tx, callback func(docId string, doc map[string]string) error) error
}

type Database struct {
	db *bolt.DB
}

func (db *Database) DeleteTable(tx *bolt.Tx, tableName string) error {
	existingBucket := tx.Bucket([]byte(tableName))
	if existingBucket == nil {
		return errors.New("table does not exist")
	}

	err := tx.DeleteBucket([]byte(tableName))
	if err != nil {
		return err
	}
	return nil
}

func (db *Database) CreateTable(
	tx *bolt.Tx,
	tableName string,
	strongConsistency bool,
) (Table, error) {
	existingBucket := tx.Bucket([]byte(tableName))
	if existingBucket != nil {
		return nil, errors.New("table already exists")
	}

	bucket, err := tx.CreateBucket([]byte(tableName))
	if err != nil {
		return nil, err
	}

	consistencyByte := byte(0)
	if strongConsistency {
		consistencyByte = byte(1)
	}
	bucket.Put([]byte("__consistency"), []byte{consistencyByte})

	if strongConsistency {
		return &RaftTable{
			Name: tableName,
			DB:   db.db,
		}, nil
	} else {
		return &CRDTTable{
			Name: tableName,
			DB:   db.db,
		}, nil
	}
}

func (db *Database) GetTable(
	tx *bolt.Tx,
	tableName string,
) (Table, error) {
	bucket := tx.Bucket([]byte(tableName))
	if bucket == nil {
		return nil, errors.New("table does not exist")
	}

	consistencyBytes := bucket.Get([]byte("__consistency"))
	if consistencyBytes == nil || len(consistencyBytes) != 1 {
		log.Panic("missing consistency flag")
	}

	strongConsistency := consistencyBytes[0] == 1
	if strongConsistency {
		return &RaftTable{
			Name: tableName,
			DB:   db.db,
		}, nil
	} else {
		return &CRDTTable{
			Name: tableName,
			DB:   db.db,
		}, nil
	}
}

func (db *Database) ForEach(
	tx *bolt.Tx,
	callback func(tableName string, table Table) error,
) error {
	return tx.ForEach(func(name []byte, bucket *bolt.Bucket) error {
		tableName := string(name)

		consistencyBytes := bucket.Get([]byte("__consistency"))
		if consistencyBytes == nil || len(consistencyBytes) != 1 {
			log.Panic("missing consistency flag")
		}
		var table Table
		strongConsistency := consistencyBytes[0] == 1
		if strongConsistency {
			table = &RaftTable{
				Name: tableName,
				DB:   db.db,
			}
		} else {
			table = &CRDTTable{
				Name: tableName,
				DB:   db.db,
			}
		}

		return callback(tableName, table)
	})
}

func (db *Database) OpenTx(callback func(tx *bolt.Tx) error) error {
	return db.db.Update(callback)
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
