package main

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

func calcularDesconto(nivel string, temDependentes bool) float64 {
	switch nivel {
	case "A":
		if temDependentes {
			return 0.08
		}
		return 0.03
	case "B":
		if temDependentes {
			return 0.1
		}
		return 0.05
	case "C":
		if temDependentes {
			return 0.15
		}
		return 0.08
	case "D":
		if temDependentes {
			return 0.17
		}
		return 0.1
	default:
		return 0.0
	}
}

func calcularSalarioLiquido(nome string, nivel string, salarioBruto float64, dependentes int) {
	desconto := calcularDesconto(nivel, dependentes > 0)
	salarioLiquido := salarioBruto - (salarioBruto * desconto)

	fmt.Printf("Nome: %s\n", nome)
	fmt.Printf("Nível: %s\n", nivel)
	fmt.Printf("Salário Líquido: R$ %.2f\n", salarioLiquido)
}

func main() {
	scanner := bufio.NewScanner(os.Stdin)

	fmt.Print("Nome: ")
	scanner.Scan()
	nome := scanner.Text()

	fmt.Print("Nível: ")
	scanner.Scan()
	nivel := scanner.Text()

	fmt.Print("Salário Bruto: ")
	scanner.Scan()
	salarioBrutoStr := scanner.Text()
	salarioBruto, err := strconv.ParseFloat(strings.TrimSpace(salarioBrutoStr), 64)
	if err != nil {
		fmt.Println("Salário Bruto inválido")
		return
	}

	fmt.Print("Número de Dependentes: ")
	scanner.Scan()
	dependentesStr := scanner.Text()
	dependentes, err := strconv.Atoi(strings.TrimSpace(dependentesStr))
	if err != nil {
		fmt.Println("Número de Dependentes inválido")
		return
	}

	calcularSalarioLiquido(nome, nivel, salarioBruto, dependentes)
}
