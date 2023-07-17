# Projeto: Banco de Dados Replicados na Edge - Uma Abordagem Híbrida

## Alunos:
- Ben Hur Faria Reis
- Filipe Silveira Chaves
- Thiago Monteles de Sousa
- Felipe Aguiar Costa
- João Paulo Rocha

**Contextualização**:
No contexto de replicação de bancos de dados, as abordagens de consistência forte ou eventual oferecem garantias diferentes, 
sendo que em alguns casos, uma ou outra é mais recomendada. Nesse sentido, 
o projeto pretende criar um banco de dados replicado de forma que duas abordagens de consistência estejam implementadas,
e com base no uso, o usuário pode escolher qual prefere utilizar.


**Objetivos:**

- Desenvolver um banco de dados replicado na edge que ofereça consistência forte e consistência eventual.
- Explorar o uso de técnicas como os Conflict-free Replicated Data Types (CRDT) para garantir a consistência eventual.
- Utilizar o algoritmo Raft para assegurar a consistência forte.
- Proporcionar aos usuários a capacidade de selecionar a abordagem de consistência mais adequada às suas necessidades e ao contexto de uso do banco de dados replicado.


## Etapas do Projeto:

### Para concluir o projeto, teremos as seguintes etapas:

1. Revisão bibliográfica: (TODOS) 
2. Elaboração de apresentações internas sobre métodos de consistência (TODOS) 
3. Decisão sobre os métodos de consistência (TODOS) 
4. Elaboração de um esquema arquitetural de comunicação e dados. (TODOS) 
5. Codificação de um banco de dados replicado (Felipe Aguiar, Ben Hur)
6. Codificação do protocolo CRDT (Felipe Aguiar, Ben Hur)
7. Codificação do protocolo RAFT (Filipe Chaves, João Paulo)
8. Codificação de um visualizador dos estados dos nós (Thiago, Ben Hur)
9. Codificação do agregador final dos protocolos e banco de dados replicado 

## Tabela de Status:

| Status        | Etapas                                        | Observação                               |
|---------------|-----------------------------------------------|------------------------------------------|
| Concluída     | 1 ao 4                                        |                                          |
| Em andamento  | 5, 6, 7, 8                                   | Todos possuem algum nível de implementação |
| Em breve      | 9                                             | Etapa final                              |

