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
		var idade uint8
		if err := binary.Read(conn, binary.BigEndian, &idade); err != nil {
			if err == io.EOF {
				log.Printf("Conexão %d fechada pelo cliente\n", id)
				return
			}
			log.Printf("Erro lendo idade: %v", err)
			return
		}

		var categoria string
		switch {
		case idade >= 5 && idade <= 7:
			categoria = "infantil A"
		case idade >= 8 && idade <= 10:
			categoria = "infantil B"
		case idade >= 11 && idade <= 13:
			categoria = "juvenil A"
		case idade >= 14 && idade <= 17:
			categoria = "juvenil B"
		default:
			categoria = "adulto"
		}

		if err := binary.Write(conn, binary.BigEndian, []byte(categoria)); err != nil {
			log.Printf("Erro escrevendo categoria: %v", err)
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
		idadeStr := scanner.Text()

		idade, err := strconv.ParseUint(strings.TrimSpace(idadeStr), 10, 8)
		if err != nil {
			fmt.Println("Idade inválida")
			continue
		}

		if err := binary.Write(conn, binary.BigEndian, uint8(idade)); err != nil {
			log.Printf("Erro escrevendo idade: %v", err)
			continue
		}

		categoriaBytes := make([]byte, 20)
		if _, err := conn.Read(categoriaBytes); err != nil {
			log.Printf("Erro lendo categoria: %v", err)
			continue
		}
		categoria := strings.TrimSpace(string(categoriaBytes))

		fmt.Printf("Idade = %d, Categoria = %s\n", idade, categoria)
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
