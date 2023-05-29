package main

import (
	"bufio"
	"encoding/binary"
	"flag"
	"fmt"
	"io"
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

func serverHandleConn(conn net.Conn, id uint) {
	defer conn.Close()

	fmt.Printf("Nova conexão %d\n", id)

	for {
		var saldo float64
		if err := binary.Read(conn, binary.BigEndian, &saldo); err != nil {
			if err == io.EOF {
				log.Printf("Conexão %d fechada pelo cliente\n", id)
				return
			}
			log.Printf("Erro lendo saldo: %v", err)
			return
		}

		var credito float64
		if saldo >= 0 && saldo <= 200 {
			credito = 0
		} else if saldo >= 201 && saldo <= 400 {
			credito = 0.2 * saldo
		} else if saldo >= 401 && saldo <= 600 {
			credito = 0.3 * saldo
		} else if saldo >= 601 {
			credito = 0.4 * saldo
		}

		if err := binary.Write(conn, binary.BigEndian, credito); err != nil {
			log.Printf("Erro escrevendo credito: %v", err)
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
	for scanner.Scan() {
		saldoStr := scanner.Text()

		saldo, err := strconv.ParseFloat(strings.TrimSpace(saldoStr), 64)
		if err != nil {
			fmt.Println("Saldo invalido")
			continue
		}

		if err := binary.Write(conn, binary.BigEndian, saldo); err != nil {
			log.Printf("Erro escrevendo saldo: %v", err)
			continue
		}

		var credito float64
		if err := binary.Read(conn, binary.BigEndian, &credito); err != nil {
			log.Printf("Erro lendo credito: %v", err)
			continue
		}

		fmt.Printf("Saldo = %f, Credito = %f\n", saldo, credito)
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
