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
	Idade        uint8
	TempoServico uint8
}

type Resp struct {
	PodeAposentar bool
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
		if err := encoder.Encode(&Resp{
			PodeAposentar: req.Idade >= 65 && req.TempoServico >= 30 && (req.Idade >= 60 && req.TempoServico >= 25),
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
		fmt.Printf("Idade: ")
		scanner.Scan()
		idadeStr := scanner.Text()

		fmt.Printf("Tempo de serviço: ")
		scanner.Scan()
		tempoServicoStr := scanner.Text()

		idade, err := strconv.ParseInt(strings.TrimSpace(idadeStr), 10, 8)
		if err != nil {
			fmt.Println("Idade inválida")
			continue
		}

		tempoServico, err := strconv.ParseInt(strings.TrimSpace(tempoServicoStr), 10, 8)
		if err != nil {
			fmt.Println("Tempo de serviço inválido")
			continue
		}

		encoder := gob.NewEncoder(conn)
		if err := encoder.Encode(&Req{
			Idade:        uint8(idade),
			TempoServico: uint8(tempoServico),
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

		fmt.Printf("Pode aposentar: %v\n\n", resp.PodeAposentar)
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
