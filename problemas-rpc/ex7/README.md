# Exercício 7

## Instalando geradores de código para o gRPC

Para go:

```bash
go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.28
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.2
```

Para python:

```bash
pip install grpcio-tools
```

## Gerando código para o gRPC

```bash
make
```

## Instalando dependências para Go

```bash
cd server
go mod tidy
```

## Instalando dependências para Python

```bash
pip install grpc
```

## Rodando o servidor em Go

```bash
go run ./server
```

## Rodando o cliente em Python

```bash
python ./client
```
