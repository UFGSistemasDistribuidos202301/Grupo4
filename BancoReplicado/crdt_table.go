package main

import (
	"banco_de_dados/crdt"
	"banco_de_dados/pb"
	"context"
	"encoding/json"
	"errors"
	"log"
	"strings"
	"time"

	bolt "go.etcd.io/bbolt"
)

type CRDTTable struct {
	DB       *bolt.DB
	Instance *Instance
	Name     string
}

func (t *CRDTTable) Put(
	tx *bolt.Tx,
	docId string,
	doc map[string]string,
) error {
	bucket := tx.Bucket([]byte(t.Name))
	if bucket == nil {
		return errors.New("table does not exist")
	}

	var crdtDoc crdt.MergeableMap
	if docBytes := bucket.Get([]byte(docId)); docBytes != nil {
		err := json.Unmarshal(docBytes, &crdtDoc)
		if err != nil {
			return err
		}

		crdtDoc = crdtDoc.SetDeleted(false)

		// Remove all values from current doc
		keys := []string{}
		for k := range crdtDoc.Map {
			keys = append(keys, k)
		}
		for _, key := range keys {
			crdtDoc = crdtDoc.Remove(key)
		}
	} else {
		log.Printf("Instance: %v\n", t.Instance)
		crdtDoc = crdt.NewMergeableMap(int(t.Instance.nodeID))
	}

	for k, v := range doc {
		crdtDoc = crdtDoc.Put(k, v)
	}

	docBytes, err := json.Marshal(crdtDoc)
	if err != nil {
		return err
	}

	err = bucket.Put([]byte(docId), docBytes)
	if err != nil {
		return err
	}

	// Queue CRDT broadcast
	t.Instance.queueCRDTState(t.Name, docId, crdtDoc)

	return nil
}

func (t *CRDTTable) Patch(
	tx *bolt.Tx,
	docId string,
	doc map[string]string,
) (map[string]string, error) {
	bucket := tx.Bucket([]byte(t.Name))
	if bucket == nil {
		return nil, errors.New("table does not exist")
	}

	var crdtDoc crdt.MergeableMap
	if docBytes := bucket.Get([]byte(docId)); docBytes != nil {
		err := json.Unmarshal(docBytes, &crdtDoc)
		if err != nil {
			return nil, err
		}
		crdtDoc = crdtDoc.SetDeleted(false)
	} else {
		crdtDoc = crdt.NewMergeableMap(int(t.Instance.nodeID))
	}

	for k, v := range doc {
		crdtDoc = crdtDoc.Put(k, v)
	}

	docBytes, err := json.Marshal(crdtDoc)
	if err != nil {
		return nil, err
	}

	err = bucket.Put([]byte(docId), docBytes)
	if err != nil {
		return nil, err
	}

	// Queue CRDT broadcast
	t.Instance.queueCRDTState(t.Name, docId, crdtDoc)

	doc = make(map[string]string)
	for k, v := range crdtDoc.Map {
		doc[k] = v.Value
	}
	return doc, nil
}

func (t *CRDTTable) Delete(tx *bolt.Tx, docId string) (map[string]string, error) {
	bucket := tx.Bucket([]byte(t.Name))
	if bucket == nil {
		return nil, errors.New("table does not exist")
	}

	docBytes := bucket.Get([]byte(docId))
	if docBytes == nil {
		return nil, errors.New("document does not exist")
	}

	var crdtDoc crdt.MergeableMap
	err := json.Unmarshal(docBytes, &crdtDoc)
	if err != nil {
		return nil, err
	}

	if crdtDoc.Deleted {
		return nil, errors.New("document does not exist")
	}

	crdtDoc = crdtDoc.SetDeleted(true)

	docBytes, err = json.Marshal(crdtDoc)
	if err != nil {
		return nil, err
	}

	err = bucket.Put([]byte(docId), docBytes)
	if err != nil {
		return nil, err
	}

	// Queue CRDT broadcast
	t.Instance.queueCRDTState(t.Name, docId, crdtDoc)

	doc := make(map[string]string)
	for k, v := range crdtDoc.Map {
		doc[k] = v.Value
	}
	return doc, nil
}

func (t *CRDTTable) Get(tx *bolt.Tx, docId string) (map[string]string, error) {
	bucket := tx.Bucket([]byte(t.Name))
	if bucket == nil {
		return nil, errors.New("table does not exist")
	}

	docBytes := bucket.Get([]byte(docId))
	if docBytes == nil {
		return nil, errors.New("document does not exist")
	}

	var crdtDoc crdt.MergeableMap
	err := json.Unmarshal(docBytes, &crdtDoc)
	if err != nil {
		return nil, err
	}

	if crdtDoc.Deleted {
		return nil, errors.New("document does not exist")
	}

	doc := make(map[string]string)
	for k, v := range crdtDoc.Map {
		doc[k] = v.Value
	}
	return doc, nil
}

func (t *CRDTTable) ForEach(
	tx *bolt.Tx,
	callback func(docId string, doc map[string]string) error,
) error {
	bucket := tx.Bucket([]byte(t.Name))
	if bucket == nil {
		return errors.New("table does not exist")
	}
	return bucket.ForEach(func(docIdBytes, docBytes []byte) error {
		docId := string(docIdBytes)
		if strings.HasPrefix(docId, "__") {
			return nil
		}

		var crdtDoc crdt.MergeableMap
		err := json.Unmarshal(docBytes, &crdtDoc)
		if err != nil {
			log.Panic(err)
		}

		if crdtDoc.Deleted {
			return nil
		}

		doc := make(map[string]string)
		for k, v := range crdtDoc.Map {
			doc[k] = v.Value
		}

		return callback(docId, doc)
	})
}

func (t *CRDTTable) Merge(
	tx *bolt.Tx,
	docId string,
	receivedCrdtDoc crdt.MergeableMap,
) error {
	bucket := tx.Bucket([]byte(t.Name))
	if bucket == nil {
		return errors.New("table does not exist")
	}

	var crdtDoc crdt.MergeableMap
	if docBytes := bucket.Get([]byte(docId)); docBytes != nil {
		err := json.Unmarshal(docBytes, &crdtDoc)
		if err != nil {
			return err
		}
	} else {
		crdtDoc = crdt.NewMergeableMap(int(t.Instance.nodeID))
	}

	crdtDoc = crdtDoc.Merge(receivedCrdtDoc)

	docBytes, err := json.Marshal(crdtDoc)
	if err != nil {
		return err
	}

	err = bucket.Put([]byte(docId), docBytes)
	if err != nil {
		return err
	}

	return nil
}

func (i *Instance) queueCRDTState(tableName string, docId string, m crdt.MergeableMap) {
	i.pendingCRDTStatesLock.Lock()
	defer i.pendingCRDTStatesLock.Unlock()

	i.pendingCRDTStates = append(i.pendingCRDTStates, &pb.DocumentCRDTState{
		TableName: tableName,
		DocId:     docId,
		Map:       m.ToPB(),
	})
}

func (i *Instance) startCRDTTimer() {
	for {
		time.Sleep(time.Second * 10)
		i.syncPendingCRDTStates()
	}
}

func (i *Instance) syncPendingCRDTStates() {
	i.pendingCRDTStatesLock.Lock()
	defer i.pendingCRDTStatesLock.Unlock()

	if len(i.pendingCRDTStates) == 0 {
		return
	}

	i.logger.Printf("Sending CRDT sync states to other nodes...\n")

	reverse(i.pendingCRDTStates)

	ctx := context.Background()

	req := &pb.MergeCRDTStatesRequest{
		Documents: i.pendingCRDTStates,
	}

	for _, client := range i.rpcClients {
		_, err := client.MergeCRDTStates(ctx, req)
		if err != nil {
			i.logger.Printf("Failed to send CRDT merge request to client: %s\n", err.Error())
		}
	}

	// Empty the slice
	i.pendingCRDTStates = i.pendingCRDTStates[:0]
}

func reverse[S ~[]E, E any](s S) {
	for i, j := 0, len(s)-1; i < j; i, j = i+1, j-1 {
		s[i], s[j] = s[j], s[i]
	}
}
