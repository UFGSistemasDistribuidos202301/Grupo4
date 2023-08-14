package main

import (
	"encoding/gob"
	"fmt"
	"log"
	"math/rand"
	"net"
	"strconv"
	"strings"
	"time"
)

var (
	nodeColors = map[int]string{
		8090: "\033[31m",
		8091: "\033[32m",
		8092: "\033[33m",
	}
)

const charset = "abcdefghijklmnopqrstuvwxyz" +
	"ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

var seededRand *rand.Rand = rand.New(rand.NewSource(time.Now().UnixNano()))

func makeRandomString(length int, charset string) string {
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[seededRand.Intn(len(charset))]
	}
	return string(b)
}

func generateRandomString(length int) string {
	return makeRandomString(length, charset)
}

func handleConnection(c net.Conn) {
	var buffer string
	dec := gob.NewDecoder(c)
	err := dec.Decode(&buffer)
	if err != nil {
		log.Fatalf("Fail to decode: %v\n", err)
	}

	info := strings.Split(buffer, "|")
	address := info[0]
	timestamp := info[1]
	port, err := strconv.Atoi(strings.Split(address, ":")[1])
	if err != nil {
		panic(err)
	}

	log.SetPrefix(nodeColors[port])
	defer log.SetPrefix("")

	fmt.Printf("\n\n")
	log.Printf("%s entered critical region with timestamp %s\n", address, timestamp)

	time.Sleep(5000 * time.Millisecond)

	enc := gob.NewEncoder(c)
	err = enc.Encode(generateRandomString(10))
	if err != nil {
		log.Fatalf("Fail to encode: %v\n", err)
	}

	log.Printf("%s left critical region with timestamp %s\n", address, timestamp)
}

func main() {
	l, err := net.Listen("tcp", "localhost:8081")

	if err != nil {
		log.Fatalln("Fail to connect")
	}

	log.Println("Server of Critical Region is running")

	defer l.Close()

	for {
		c, err := l.Accept()
		if err != nil {
			log.Fatalln("Fail to connect")
		}

		handleConnection(c)
	}
}
