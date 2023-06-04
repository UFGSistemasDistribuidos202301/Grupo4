package main

import (
	"bufio"
	"encoding/gob"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"strconv"
	"strings"
)

const (
	addr = "localhost:4444"
)

var (
	serverMode     bool
	concurrentMode bool
	connCount      uint = 0
)

type Req struct {
	Sexo        string
	Altura 		float32
}

type Resp struct {
	PesoIdeal float32
}

func serverHandleConn(conn net.Conn, id uint) {
	defer conn.Close()

	fmt.Printf("Nova conexão %d\n", id)

	for {
		decoder := gob.NewDecoder(conn)
		var req Req
		if err := decoder.Decode(&req); err != nil {
			log.Printf("Erro lendo pedido: %v", err)
			return
		}

		encoder := gob.NewEncoder(conn)
		var pesoideal float32
		if req.Sexo == "M" {
			pesoideal = 72.7*req.Altura-58
		}else{
			pesoideal = 62.1*req.Altura-44.7
		}
		if err := encoder.Encode(&Resp{
			PesoIdeal: pesoideal,
		}); err != nil {
			log.Printf("Erro escrevendo resposta: %v", err)
			return
		}

		fmt.Printf("Resposta enviada para conexão %d\n", id)
	}
}

func server() {
	fmt.Println("Rodando o servidor")

	listener, err := net.Listen("tcp", addr)
	if err != nil {
		panic(err)
	}

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("Erro aceitando a conexão: %v", err)
			continue
		}

		id := connCount
		connCount++

		if concurrentMode {
			go serverHandleConn(conn, id)
		} else {
			serverHandleConn(conn, id)
		}
	}
}

func client() {
	fmt.Println("Rodando o cliente")

	conn, err := net.Dial("tcp", addr)
	if err != nil {
		panic(err)
	}

	scanner := bufio.NewScanner(os.Stdin)
	for  {
		fmt.Printf("Sexo: [M/F] ")
		scanner.Scan()
		sexoStr := scanner.Text()

		fmt.Printf("Altura: ")
		scanner.Scan()
		alturaStr := scanner.Text()
		

		altura, err := strconv.ParseFloat(strings.TrimSpace(alturaStr), 32)
		if err != nil {
			fmt.Println("Tempo de serviço inválido")
			continue
		}

		encoder := gob.NewEncoder(conn)
		if err := encoder.Encode(&Req{
			Sexo:        sexoStr,
			Altura: float32(altura),
		}); err != nil {
			log.Printf("Erro escrevendo pedido: %v", err)
			continue
		}

		var resp Resp
		decoder := gob.NewDecoder(conn)
		if err := decoder.Decode(&resp); err != nil {
			log.Printf("Erro lendo resposta: %v", err)
			continue
		}

		fmt.Printf("Peso ideal: %v\n\n", resp.PesoIdeal)
	}
}

func main() {
	flag.BoolVar(&serverMode, "server", false, "Rodar como servidor")
	flag.BoolVar(&concurrentMode, "concurrent", false, "Rodar em modo concorrente")
	flag.Parse()

	if serverMode {
		server()
	} else {
		client()
	}
}
