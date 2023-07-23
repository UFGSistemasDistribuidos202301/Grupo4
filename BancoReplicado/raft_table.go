package main

import (
	"encoding/json"
	"errors"
	"log"
	"strings"

	bolt "go.etcd.io/bbolt"
)

type RaftTable struct {
	DB   *bolt.DB
	Name string
}

func (t *RaftTable) Put(tx *bolt.Tx, docId string, doc map[string]string) error {
	bucket := tx.Bucket([]byte(t.Name))
	if bucket == nil {
		return errors.New("table does not exist")
	}

	value, err := json.Marshal(doc)
	if err != nil {
		return err
	}

	err = bucket.Put([]byte(docId), value)
	if err != nil {
		return err
	}

	return nil
}

func (t *RaftTable) Patch(
	tx *bolt.Tx,
	docId string,
	doc map[string]string,
) (map[string]string, error) {
	bucket := tx.Bucket([]byte(t.Name))
	if bucket == nil {
		return nil, errors.New("table does not exist")
	}

	existingDoc := map[string]string{}
	existingDocBytes := bucket.Get([]byte(docId))
	if existingDocBytes != nil {
		err := json.Unmarshal(existingDocBytes, &existingDoc)
		if err != nil {
			return nil, err
		}
	}

	for k, v := range doc {
		existingDoc[k] = v
	}

	docBytes, err := json.Marshal(existingDoc)
	if err != nil {
		return nil, err
	}
	err = bucket.Put([]byte(docId), docBytes)
	if err != nil {
		return nil, err
	}

	return existingDoc, nil
}

func (t *RaftTable) Delete(tx *bolt.Tx, docId string) (map[string]string, error) {
	bucket := tx.Bucket([]byte(t.Name))
	if bucket == nil {
		return nil, errors.New("table does not exist")
	}

	docBytes := bucket.Get([]byte(docId))
	if docBytes == nil {
		return nil, errors.New("document does not exist")
	}

	if err := bucket.Delete([]byte(docId)); err != nil {
		return nil, err
	}

	var doc map[string]string
	if err := json.Unmarshal(docBytes, &doc); err != nil {
		return nil, err
	}

	return doc, nil
}

func (t *RaftTable) Get(tx *bolt.Tx, docId string) (map[string]string, error) {
	bucket := tx.Bucket([]byte(t.Name))
	if bucket == nil {
		return nil, errors.New("table does not exist")
	}
	docBytes := bucket.Get([]byte(docId))
	if docBytes == nil {
		return nil, errors.New("document does not exist")
	}

	var doc map[string]string
	err := json.Unmarshal(docBytes, &doc)
	if err != nil {
		return nil, err
	}
	return doc, nil
}

func (t *RaftTable) ForEach(
	tx *bolt.Tx,
	callback func(k string, v map[string]string) error,
) error {
	bucket := tx.Bucket([]byte(t.Name))
	if bucket == nil {
		log.Panic("unexpected bucket == nil")
	}

	return bucket.ForEach(func(docIdBytes, docBytes []byte) error {
		docId := string(docIdBytes)
		if strings.HasPrefix(docId, "__") {
			return nil
		}

		var doc map[string]string
		err := json.Unmarshal(docBytes, &doc)
		if err != nil {
			log.Panic(err)
		}

		return callback(docId, doc)
	})
}
