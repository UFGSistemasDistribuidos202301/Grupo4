package main

import (
	"html/template"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

// Node representa um nó na rede CRDT
type Node struct {
	ID     string
	Status string
}

// ObterNodos retorna a lista de nós da rede CRDT (exemplo simulado)
func ObterNodos() []Node {
	// Lógica para recuperar os nós da rede CRDT
	// Aqui está um exemplo estático
	return []Node{
		{ID: "1", Status: "leader"},
		{ID: "2", Status: "follower"},
		{ID: "3", Status: "inactive"},
	}
}

func main() {
	router := gin.Default()

	// Rota para renderizar a página com a visualização dos nós
	router.GET("/", func(c *gin.Context) {
		nodos := ObterNodos()
		c.HTML(http.StatusOK, "index.html", gin.H{
			"Nodes": nodos,
		})
	})

	// Rota para fornecer os dados dos nós como uma API JSON
	router.GET("/api/nodes", func(c *gin.Context) {
		nodos := ObterNodos()
		c.JSON(http.StatusOK, nodos)
	})

	// Carregar o arquivo HTML do modelo
	htmlTemplate := template.Must(template.ParseFiles("index.html"))

	router.SetHTMLTemplate(htmlTemplate)

	// Executar o servidor na porta 8080
	if err := router.Run(":8080"); err != nil {
		log.Fatal(err)
	}
}
