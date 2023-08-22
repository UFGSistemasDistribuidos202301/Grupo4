# Banco de Dados Replicado

## Instruções

Para rodar 10 nós (quantidade padrão):
```
make run
```

Cada nó rodará um servidor HTTP na porta `3000 + ID do nó`.

Para acessar a visualização use o endereço:
```
http://localhost:3001
```

## Arquitetura

A estrutura do banco de dados é composta por *tabelas*.

Cada tabela pode possuir um número de *documentos*.

Cada documento é um mapa de `string -> string`, por exemplo: `{"campo_a": "valor 1", "campo_b": "valor 2"}`

Como exemplo, a estrutura do banco fica da seguinte forma:
```json
{
    "tabela_a": {
        "documento_a": {
            {
                "campo_a": "valor_a",
                "campo_b": "valor_b"
            }
        },
        "documento_b": {
            {
                "campo_c": "valor_c"
            }
        },
        "documento_c": {}
    },
    "tabela_b": {}
}
```

### Sincronização de estados com CRDTs

É implementada uma sincronização de CRDTs baseada em estados.

Cada nó, ao receber uma operação de um cliente, vai enfileirando os estados dos documentos que precisam ser propagados para os outros nós.

Um timer é utilizado para propagar esses estados periodicamente, utilizando RPC multicast. (Essa estratégia poderia ser otimizada utilizando um protocolo gossip para tornar o multicast mais eficiente.)

Quando um nó recebe um novo estado para um documento, ele executa a operação `local_doc_state = merge(local_doc_state, new_doc_state)`.

Como a operação `merge` é comutativa, todo nó sempre alcança o mesmo estado final após o recebimento dos estados remotos.
