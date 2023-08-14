package main

import (
	"encoding/gob"
	"fmt"
	"log"
	"net"
	"sync"
	"time"
)

type server struct {
	sync.Mutex
	data map[string]bool
}

func (s *server) handle_connection(c net.Conn) {
	log.Println("received a request")
	var buffer string
	dec := gob.NewDecoder(c)
	err := dec.Decode(&buffer)

	if err != nil {
		log.Fatal(err)
		log.Fatal("Fail to Decode")
	}

	log.Println("decoded a request")
	log.Println("Received: ", buffer)

	s.Lock()
	s.data[buffer] = true
	s.Unlock()

	time.Sleep(3 * time.Second)

	keys := make([]string, 0)
	for key := range s.data {
		keys = append(keys, key)
	}

	enc := gob.NewEncoder(c)
	err = enc.Encode(keys)

	if err != nil {
		log.Fatal("Fail to Encode")
	}
}

func (s *server) run_server() {
	s.data = make(map[string]bool)
	l, err := net.Listen("tcp", "localhost:8080")

	if err != nil {
		log.Fatal("Fail to connect")
	}

	log.Println("Server Running")

	defer l.Close()

	fmt.Println("Ready to receive requests")

	go func() {
		for {
			c, err := l.Accept()

			if err != nil {
				log.Fatal("Fail to connect")
			}

			go s.handle_connection(c)
		}
	}()

	ch := make(chan int)
	<-ch
}

func main() {
	name_server := server{}
	name_server.run_server()
}
