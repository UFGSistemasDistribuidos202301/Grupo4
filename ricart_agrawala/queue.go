package main

import (
	"sync"
)

type messageQueue struct {
	mutex  sync.Mutex
	buffer []message
}

func NewMessageQueue() messageQueue {
	return messageQueue{buffer: make([]message, 0)}
}

func (q *messageQueue) push(msg message) {
	q.mutex.Lock()
	q.buffer = append(q.buffer, msg)
	q.mutex.Unlock()
}

func (q *messageQueue) pop() message {
	q.mutex.Lock()
	msg := q.buffer[0]
	q.buffer = q.buffer[1:]
	q.mutex.Unlock()
	return msg
}

func (q *messageQueue) empty() bool {
	return len(q.buffer) == 0
}

func (q *messageQueue) size() int {
	return len(q.buffer)
}
