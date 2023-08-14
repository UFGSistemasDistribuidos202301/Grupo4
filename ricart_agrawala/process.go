package main

import (
	"encoding/gob"
	"fmt"
	"log"
	"net"
	"os"
	"strconv"
	"sync"
)

type process struct {
	mutex              sync.Mutex
	address            string
	id                 int
	state              int
	timestamp          int
	requestTimestamp   int
	processesAddresses []string
	s                  replyCounter
	q                  messageQueue
	channels           []chan message
	receivedAllReplies chan bool
	channelIndex       map[string]int
	logger             *log.Logger
}

var (
	nodeColors = map[int]string{
		8090: "\033[31m",
		8091: "\033[32m",
		8092: "\033[33m",
	}
)

func newProcess(port int) process {
	return process{
		logger: log.New(
			os.Stderr,
			fmt.Sprintf("%s[NODE %d] ", nodeColors[port], port),
			log.LstdFlags,
		),
		id:                 port,
		address:            fmt.Sprintf("localhost:%d", port),
		s:                  NewReplyCounter(),
		q:                  NewMessageQueue(),
		channelIndex:       make(map[string]int),
		receivedAllReplies: make(chan bool),
	}
}

func (p *process) getState() string {
	state := ""
	if p.state == WANTED {
		state = "WANTED"
	}

	if p.state == RELEASED {
		state = "RELEASED"
	}

	if p.state == HELD {
		state = "HELD"
	}
	return state
}

func (p *process) numberOfProcesses() int {
	return len(p.processesAddresses)
}

func (p *process) incrementReply() {
	p.updateTimestamp(p.timestamp)
	p.s.addReply()
	if p.s.size() == p.numberOfProcesses()-1 {
		p.receivedAllReplies <- true
	}
}

func (p *process) clearReplyCounter() {
	p.updateTimestamp(p.timestamp)
	p.s.clear()
}

func (p *process) enqueueMessage(msg message) {
	p.updateTimestamp(p.timestamp)
	p.q.push(msg)
}

func (p *process) getMessageQueueTopRequest() string {
	p.updateTimestamp(p.timestamp)
	return p.q.pop().Address
}

func (p *process) isMessageQueueEmpty() bool {
	p.updateTimestamp(p.timestamp)
	return p.q.empty() == true
}

func (p *process) updateTimestamp(timestamp int) {
	p.mutex.Lock()
	p.timestamp = max(p.timestamp, timestamp) + 1
	p.mutex.Unlock()
}

func (p *process) changeState(state int) {
	p.updateTimestamp(p.timestamp)
	p.mutex.Lock()
	p.state = state
	p.mutex.Unlock()
}

func (p *process) updateRequestTimestamp() {
	p.requestTimestamp = p.timestamp
}

func (p *process) getIndexFromAddress(address string) int {
	return p.channelIndex[address]
}

func (p *process) startProcess() {
	p.logger.Printf("Starting process...")

	p.changeState(RELEASED)

	if err := p.startListenPort(); err != nil {
		p.logger.Fatal("Error on startListenPort")
	}

	if err := p.getOtherProcessesAddresses(); err != nil {
		p.logger.Fatal("Error on getOtherProcessesAddresses")
	}

	if err := p.openAllProcessesTCPConnections(); err != nil {
		p.logger.Fatal("Error on openAllProcessesTCPConnections")
	}

	p.logger.Println("Process ", p.id, " is ready")
}

func (p *process) sendPermissionToAllProcesses() {
	p.doMulticast(PERMISSION)
	p.waitAllProcessesReplies()
	p.clearReplyCounter()
}

func (p *process) openTCPConnection(address string, TCPWaiter chan bool) error {
	//open TCP connection on address
	connection, err := net.Dial("tcp", address)
	if err != nil {
		p.logger.Println("Error in opening TCP port: ", address)
	}
	TCPWaiter <- true

	go func(address string, connection net.Conn) {
		var msg message

		//create encoder serializer
		encoder := gob.NewEncoder(connection)
		defer connection.Close()

		for {
			// channel waiting some message
			msg = <-p.channels[p.getIndexFromAddress(address)]
			p.logger.Printf("%d Process %d is sending a %s with timestamp %d to %s\n", p.timestamp, p.id, msg.getType(), msg.Timestamp, address)

			//sending message
			if err := msg.encodeAndSendMessage(encoder); err != nil {
				p.logger.Println(p.timestamp, " Error on Process ", p.id)
				p.logger.Println(err)
			}
		}
	}(address, connection)

	return nil
}

func (p *process) openAllProcessesTCPConnections() error {
	//creating slice of channels
	p.channels = make([]chan message, p.numberOfProcesses())

	//channel to wait all the TCP
	TCPWaiter := make(chan bool, p.numberOfProcesses()-1)
	for i, address := range p.processesAddresses {

		p.channelIndex[address] = i

		//creating each channel
		p.channels[i] = make(chan message)

		if address != p.address {
			if err := p.openTCPConnection(address, TCPWaiter); err != nil {
				return err
			}
		}
	}

	//waiting all TCP connections to be ready
	for i := 0; i < p.numberOfProcesses()-1; i++ {
		<-TCPWaiter
	}
	return nil

}

func (p *process) sendMessage(typeMessage int, address string) {
	//creating new message
	msg := message{
		Timestamp:        p.timestamp,
		RequestTimestamp: p.requestTimestamp,
		TypeMessage:      typeMessage,
		Address:          p.address,
		Id:               p.id,
	}

	// sending message to address's channel
	p.channels[p.getIndexFromAddress(address)] <- msg
}

func (p *process) doMulticast(typeMessage int) {
	p.updateTimestamp(p.timestamp)

	//send message to all processes
	for _, address := range p.processesAddresses {
		if address != p.address {
			go p.sendMessage(typeMessage, address)
		}
	}
}

// Method related to servers
func (p *process) getOtherProcessesAddresses() error {

	conn, err := net.Dial("tcp", REGISTER_SERVER_ADDRESS)
	if err != nil {
		return err
	}

	defer conn.Close()

	enc := gob.NewEncoder(conn)
	if err := enc.Encode(p.address); err != nil {
		return err
	}

	dec := gob.NewDecoder(conn)
	if err = dec.Decode(&p.processesAddresses); err != nil {
		return err
	}

	return nil
}

// Method related to server
func (p *process) getRandomStringFromServer() {
	conn, err := net.Dial("tcp", CRITICAL_REGION_SERVER_ADDRESS)

	defer conn.Close()

	if err != nil {
		p.logger.Println("Dial on ", p.address)
		p.logger.Fatal(err)
	}

	msg := p.address + "|" + strconv.Itoa(p.timestamp)

	enc := gob.NewEncoder(conn)
	err = enc.Encode(msg)
	if err != nil {
		p.logger.Fatalf("Error encoding message: %v\n", err)
	}

	dec := gob.NewDecoder(conn)
	err = dec.Decode(&msg)
	if err != nil {
		p.logger.Fatalf("Error decoding message: %v\n", err)
	}

	p.logger.Printf("%d Process %d received random string from critical region server: %s\n", p.timestamp, p.id, msg)
}
