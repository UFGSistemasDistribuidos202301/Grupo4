package main

import (
	"encoding/gob"
	"math/rand"
	"net"
	"time"
)

func (p *process) waitAllProcessesReplies() {
	<-p.receivedAllReplies
}

func (p *process) replyAllEnqueuedRequests() {
	for p.isMessageQueueEmpty() == false {
		address := p.getMessageQueueTopRequest()
		p.logger.Printf("%d Process %d send ENQUEUED REPLY to %s\n", p.timestamp, p.id, address)
		p.sendMessage(REPLY, address)
	}
}

func (p *process) startListenPort() error {
	//opening TCP port
	listener, err := net.Listen("tcp", p.address)
	if err != nil {
		return err
	}

	go func(listener net.Listener) {
		defer listener.Close()
		for {
			conn, err := listener.Accept()
			if err != nil {
				p.logger.Println(err)
			}
			//handling new connection
			go p.handleRequest(conn)
		}
	}(listener)
	return nil
}

func (p *process) handleRequest(connection net.Conn) {
	defer connection.Close()

	//creating decoder serializer
	decoder := gob.NewDecoder(connection)

	for {
		var msg message
		// blocked waiting some message
		if err := msg.receiveAndDecodeMessage(decoder); err != nil {
			p.logger.Println("error on receiveAndDecodeMessage")
			p.logger.Fatal(err)
		}

		p.logger.Printf(
			"%d Process %d on state %s received a %s from %d  address: %s with timestamp %d size: %d\n",
			msg.Timestamp,
			p.id,
			p.getState(),
			msg.getType(),
			msg.Id,
			msg.Address,
			msg.Timestamp,
			p.s.size(),
		)

		p.updateTimestamp(msg.Timestamp)
		// described in book
		if msg.TypeMessage == REPLY || msg.TypeMessage == PERMISSION {
			p.incrementReply()
		} else {
			if p.state == HELD || (p.state == WANTED && less(p, msg)) {
				p.logger.Printf(
					"%d Process %d on state %s enqueued because %d is less than %d size counter: %d queue size: %d\n",
					p.timestamp,
					p.id,
					p.getState(),
					p.requestTimestamp,
					msg.RequestTimestamp,
					p.s.size(),
					p.q.size(),
				)
				p.enqueueMessage(msg)
			} else {
				p.logger.Printf(
					"%d Process %d on state %s is sending a reply to %s because %d is bigger than %d size counter: %d queue size: %d\n",
					p.timestamp,
					p.id,
					p.getState(),
					msg.Address,
					p.requestTimestamp,
					msg.RequestTimestamp,
					p.s.size(),
					p.q.size(),
				)
				p.sendMessage(REPLY, msg.Address)
			}
		}
	}
}

func (p *process) enterOnCriticalSection() {
	//described on book
	p.changeState(WANTED)
	p.updateRequestTimestamp()
	p.doMulticast(REQUEST)
	p.waitAllProcessesReplies()
	p.changeState(HELD)
}

func (p *process) leaveCriticalSection() {
	//described on book
	p.changeState(RELEASED)
	p.replyAllEnqueuedRequests()
	p.clearReplyCounter()
	p.updateTimestamp(p.timestamp)
}

func (p *process) runRicartAgrawala() {
	p.logger.Printf("%d Process %d is running %s\n", p.timestamp, p.id, p.address)
	p.sendPermissionToAllProcesses()

	var seededRand *rand.Rand = rand.New(rand.NewSource(time.Now().UnixNano()))
	// process running entering and leaving the critical region
	for {
		p.logger.Printf("%d Process %d IS TRYING TO ENTER IN CRITICAL REGION\n", p.timestamp, p.id)

		p.enterOnCriticalSection()
		p.logger.Printf("%d Process %d ENTERED CRITICAL REGION\n", p.timestamp, p.id)

		p.getRandomStringFromServer() // Use critical region server

		p.leaveCriticalSection()
		p.logger.Printf("%d Process %d LEFT CRITICAL REGION\n", p.timestamp, p.id)
		time.Sleep(time.Duration(seededRand.Intn(500)) * time.Millisecond)
	}
}
