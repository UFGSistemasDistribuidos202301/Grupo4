package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"strconv"
	"strings"
)

const (
	addr = "localhost:4446"
)

var (
	serverMode     bool
	concurrentMode bool
	connCount      uint = 0
)

func serverHandleConn(conn net.Conn, id uint) {  // Função que trata a conexão com o cliente
	defer conn.Close() // Fecha a conexão quando a função terminar

	fmt.Printf("Nova conexão %d\n", id)

	for {

				// Read from socket to buffer
		buffer := make([]byte, 1024)
		mLen, err := conn.Read(buffer)
		if err != nil {
			fmt.Printf("Failed to read from socket!\n")
			conn.Close()
			return
		}

		// Check if message is valid then treat
		message := string(buffer[:mLen])
		if strings.Compare(message[:3], "RAJ") == 0 { // checa se o inicio da mensagem é RAJ
			data := strings.Split(message[4:], " ") 		// separa a mensagem em um array de strings
			salary, _ := strconv.ParseFloat(data[2], 64) 	// converte o salario para float64


			// checa o cargo e aumenta o salario
			switch data[1] {
			case "operador":
				salary *= 1.2
			case "programador":
				salary *= 1.18
			}

			_, err := conn.Write([]byte(fmt.Sprintf("%f", salary))) // escreve no socket o salario
			if err != nil {
				fmt.Printf("Failed to write to socket!\n")
				conn.Close()
				return
			}

		} else {
			fmt.Printf("Invalid message!")
		}
		
	}
}

func server() {
	fmt.Println("Rodando o servidor")

	listener, err := net.Listen("tcp", addr)  // Cria um listener para o endereço especificado
	if err != nil {
		panic(err)
	}

	for {
		conn, err := listener.Accept() 		// Aceita uma conexão
		
		if err != nil {
			log.Printf("Erro aceitando a conexão: %v", err)
			continue
		}

		id := connCount
		connCount++

		if concurrentMode {  
			go serverHandleConn(conn, id)  // Executa a função serverHandleConn em uma goroutine separada
		} else {
			serverHandleConn(conn, id)
		}
	}
}

func client() {
	fmt.Println("Rodando o cliente")

	conn, err := net.Dial("tcp", addr)  // Estabelece uma conexão com o servidor
	if err != nil {
		panic(err)
	}

	fmt.Print("-------------------- Ajuste salarial --------------------\n") 
	fmt.Print("formato de entrada: NOME CARGO SALARIO\n")
	fmt.Print("type 'exit' to close\n")

	reader := bufio.NewReader(os.Stdin)


	for  {
		 // Collect input data from terminal
		 fmt.Print("> ")
		 input, _ := reader.ReadString('\n')
		 input = strings.Replace(input, "\n", "", -1)
		 input = strings.Replace(input, "\r", "", -1)
 
		 if strings.Compare(input, "exit") == 0 {
			 break
		 }
 
		 // Check that the input string has the correct format
		 data := strings.Split(input, " ")
		 if len(data)%3 != 0 {
			 panic("Invalid input!")
		 }
 
		 // Counts the number o employees present in the input data.
		 numEmployees := len(data) / 3
		 for i := 0; i < numEmployees; i++ {
			 // Checks if salary value is valid.
			 if _, err := strconv.ParseFloat(data[(i*3)+2], 64); err != nil {
				 panic("Invalid input!")
			 }


			var name string = data[(i*3)]
			var role string = data[(i*3)+1]
			var salary string = data[(i*3)+2]

			// Make readjustment request
			_, err = conn.Write([]byte(fmt.Sprintf("RAJ %s %s %s", name, role, salary)))
			if err != nil {
				panic("Failed to write to socket!")
			}

			// Receive results
			buffer := make([]byte, 8)
			mLen, err := conn.Read(buffer)
			if err != nil {
				panic("Failed to read from socket!")
			}

			// Present results
			oldSal, _ := strconv.ParseFloat(salary, 64)
			newSal, _ := strconv.ParseFloat(string(buffer[:mLen]), 64)
			fmt.Printf("NAME: %s | ROLE: %s | SALARY: %.2f -> %.2f\n", name, role, oldSal, newSal)
		 }
	}

}


func main(){
	// Configuração da flag -server e -concurrent para definir o modo de execução como servidor
	flag.BoolVar(&serverMode, "server", false, "Rodar como servidor")
	flag.BoolVar(&concurrentMode, "concurrent", false, "Rodar em modo concorrente")
	// Analisa as flags fornecidas na linha de comando
	flag.Parse()

	
	if serverMode {
		server()
	} else {
		client()
	}
}
