package main

import (
	"banco_de_dados/crdt"
	"banco_de_dados/pb"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"go.etcd.io/bbolt"
	bolt "go.etcd.io/bbolt"
	"google.golang.org/protobuf/proto"
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
	go t.Instance.queueCRDTStateForAllNodes(t.Name, docId, crdtDoc)

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
	go t.Instance.queueCRDTStateForAllNodes(t.Name, docId, crdtDoc)

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
	go t.Instance.queueCRDTStateForAllNodes(t.Name, docId, crdtDoc)

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
	for j := *baseNodeID; j < *baseNodeID+*nodeCount; j++ {
		if j == i.nodeID {
			continue
		}

		_, err := i.QueuePendingCRDTStates(context.Background(), &pb.QueuePendingCRDTStatesRequest{
			DestNodeID: uint64(j),
			Documents: []*pb.DocumentCRDTState{
				{
					TableName: tableName,
					DocId:     docId,
					Map:       m.ToPB(),
					IsRetry:   false,
				},
			},
		})
		if err != nil {
			i.logger.Printf("Failed to queue CRDT state: %s\n", err.Error())
		}
	}
}

func (i *Instance) startCRDTTimer() {
	counter := 10
	for {
		time.Sleep(time.Second * 1)

		if !i.isOffline() && i.isCRDTTimerEnabled() {
			counter -= 1

			if counter <= 0 {
				counter = 10
				i.syncPendingCRDTStates()
			}

			i.visEventsChannel <- VisEvent{
				NodeID: i.nodeID,
				Kind:   "timer",
				Data:   counter,
			}
		}
	}
}

func (i *Instance) getPendingCRDTStatesForNode(nodeID uint) []*pb.DocumentCRDTState {
	bucketName := []byte(fmt.Sprintf("__crdt_pending_states_%d", nodeID))

	var pendingStates []*pb.DocumentCRDTState
	err := i.db.OpenReadOnlyTx(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(bucketName)
		if bucket == nil {
			return nil
		}

		bucket.ForEach(func(k, v []byte) error {
			state := &pb.DocumentCRDTState{}
			err := proto.Unmarshal(k, state)
			if err != nil {
				return err
			}
			pendingStates = append(pendingStates, state)
			return nil
		})

		return nil
	})
	if err != nil {
		i.logger.Printf("Failed to get pending CRDT states: %s\n", err.Error())
		return nil
	}
	return pendingStates
}

func (i *Instance) flushPendingCRDTStatesForNode(nodeID uint) error {
	bucketName := []byte(fmt.Sprintf("__crdt_pending_states_%d", nodeID))

	err := i.db.OpenTx(func(tx *bolt.Tx) error {
		err := tx.DeleteBucket(bucketName)
		if err != bbolt.ErrBucketNotFound {
			return err
		}
		return nil
	})
	if err != nil {
		i.logger.Printf("Failed to flush pending CRDT states: %s\n", err.Error())
		return err
	}
	return nil
}

func (i *Instance) syncPendingCRDTStatesWithNode(nodeID uint) {
	pendingStates := i.getPendingCRDTStatesForNode(nodeID)
	if len(pendingStates) == 0 {
		return
	}

	i.logger.Printf("Sending CRDT merge request to node %d...\n", nodeID)
	i.flushPendingCRDTStatesForNode(nodeID)
	ctx := context.Background()

	rpcClient := i.rpcClients[nodeID]
	_, err := rpcClient.MergeCRDTStates(ctx, &pb.MergeCRDTStatesRequest{
		Documents: pendingStates,
	})
	if err != nil {
		i.logger.Printf(
			"Failed to send CRDT merge request to node %d: %s (pending states = %d)\n",
			nodeID,
			err.Error(),
			len(pendingStates),
		)

		_, err = i.QueuePendingCRDTStates(
			ctx,
			&pb.QueuePendingCRDTStatesRequest{
				DestNodeID: uint64(nodeID),
				Documents:  pendingStates,
			},
		)
		if err != nil {
			i.logger.Printf("Failed to queue CRDT states for retry for node %d at node %d: %s\n", nodeID, i.nodeID, err.Error())
		}

		retries := []*pb.DocumentCRDTState{}
		for _, state := range pendingStates {
			if !state.IsRetry {
				retries = append(retries, state)
				state.IsRetry = true
			}
		}

		if len(retries) > 0 {
			for j := *baseNodeID; j < *baseNodeID+*nodeCount; j++ {
				if j != i.nodeID {
					_, err = i.rpcClients[j].QueuePendingCRDTStates(
						ctx,
						&pb.QueuePendingCRDTStatesRequest{
							DestNodeID: uint64(nodeID),
							Documents:  retries,
						},
					)
					if err != nil {
						i.logger.Printf("Failed to queue CRDT states for retry for node %d: %s\n", nodeID, err.Error())
					}
				}
			}
		}
	} else {
		i.logger.Printf("Sent CRDT merge request to node %d\n", nodeID)
	}
}

func (i *Instance) syncPendingCRDTStates() {
	if i.isOffline() {
		return
	}

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
