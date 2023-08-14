package main

import (
	"sync"
)

type replyCounter struct {
	mutex   sync.Mutex
	counter int
}

func NewReplyCounter() replyCounter {
	return replyCounter{counter: 0}
}

func (s *replyCounter) addReply() {
	s.mutex.Lock()
	s.counter += 1
	s.mutex.Unlock()
}

func (s replyCounter) size() int {
	return s.counter
}

func (s *replyCounter) clear() {
	s.mutex.Lock()
	s.counter = 0
	s.mutex.Unlock()
}
