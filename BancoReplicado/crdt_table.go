package main

import (
	"banco_de_dados/crdt"
	"time"

	bolt "go.etcd.io/bbolt"
)

var (
	pendingCRDTStates = make(chan *crdt.MergeableMap)
)

type CRDTTable struct {
	DB   *bolt.DB
	Name string
}

func (t *CRDTTable) Put(
	tx *bolt.Tx,
	docId string,
	doc map[string]string,
) error {
	panic("unimplemented")
}

func (t *CRDTTable) Patch(
	tx *bolt.Tx,
	docId string,
	doc map[string]string,
) (map[string]string, error) {
	panic("unimplemented")
}

func (t *CRDTTable) Delete(tx *bolt.Tx, docId string) (map[string]string, error) {
	panic("unimplemented")
}

func (t *CRDTTable) Get(tx *bolt.Tx, docId string) (map[string]string, error) {
	panic("unimplemented")
}

func (t *CRDTTable) ForEach(
	tx *bolt.Tx,
	callback func(docId string, doc map[string]string) error,
) error {
	panic("unimplemented")
}

func startCRDTTimer() {
	for {
		time.Sleep(time.Second * 10)
		syncPendingCRDTStates()
	}
}

func syncPendingCRDTStates() {
	for _, client := range rpcClients {
		_ = client
		// client.MergeCRDTStates()
	}
}
