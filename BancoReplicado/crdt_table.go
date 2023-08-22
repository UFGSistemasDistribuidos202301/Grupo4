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
	t.Instance.queueCRDTStateForAllNodes(t.Name, docId, crdtDoc)

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
	t.Instance.queueCRDTStateForAllNodes(t.Name, docId, crdtDoc)

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
	t.Instance.queueCRDTStateForAllNodes(t.Name, docId, crdtDoc)

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

func (i *Instance) queueCRDTStateForAllNodes(tableName string, docId string, m crdt.MergeableMap) {
	i.pendingCRDTStatesLock.Lock()
	defer i.pendingCRDTStatesLock.Unlock()

	for j := *baseNodeID; j < *baseNodeID+*nodeCount; j++ {
		if j == i.nodeID {
			continue
		}

		_, err := i.QueuePendingCRDTState(context.Background(), &pb.QueuePendingCRDTStateRequest{
			NodeID: uint64(j),
			Document: &pb.DocumentCRDTState{
				TableName: tableName,
				DocId:     docId,
				Map:       m.ToPB(),
			},
		})
		if err != nil {
			i.logger.Printf("Failed to queue CRDT state: %s\n", err.Error())
		}
	}
}

func (i *Instance) startCRDTTimer() {
	for {
		time.Sleep(time.Second * 10)
		i.syncPendingCRDTStates()
	}
}

func (i *Instance) syncPendingCRDTStatesWithNode(nodeID uint) {
	i.pendingCRDTStatesLock.Lock()
	defer i.pendingCRDTStatesLock.Unlock()

	pendingStates := i.pendingCRDTStates[nodeID]
	if len(pendingStates) == 0 {
		return
	}

	reverse(pendingStates)

	ctx := context.Background()

	req := &pb.MergeCRDTStatesRequest{
		Documents: pendingStates,
	}

	rpcClient := i.rpcClients[nodeID]
	_, err := rpcClient.MergeCRDTStates(ctx, req)
	if err != nil {
		i.logger.Printf("Failed to send CRDT merge request to client: %s\n", err.Error())

		for j := *baseNodeID; j < *baseNodeID+*nodeCount; j++ {
			if j == i.nodeID {
				continue
			}

			for _, state := range pendingStates {
				_, err := i.rpcClients[j].QueuePendingCRDTState(ctx, &pb.QueuePendingCRDTStateRequest{
					NodeID:   uint64(nodeID),
					Document: state,
				})
				if err != nil {
					i.logger.Printf("Failed to queue CRDT state on remote node: %s\n", err.Error())
				}
			}
		}
	} else {
		// Empty the slice
		i.pendingCRDTStates[nodeID] = pendingStates[:0]
	}
}

func (i *Instance) syncPendingCRDTStates() {
	if i.isOffline() {
		return
	}

	i.pendingCRDTStatesLock.Lock()
	defer i.pendingCRDTStatesLock.Unlock()

	i.logger.Printf("Sending CRDT sync states to other nodes...\n")

	for j := *baseNodeID; j < *baseNodeID+*nodeCount; j++ {
		if j == i.nodeID {
			continue
		}
		go i.syncPendingCRDTStatesWithNode(j)
	}
}

func reverse[S ~[]E, E any](s S) {
	for i, j := 0, len(s)-1; i < j; i, j = i+1, j-1 {
		s[i], s[j] = s[j], s[i]
	}
}
