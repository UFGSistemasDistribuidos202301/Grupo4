import grpc
import rpc_pb2
import rpc_pb2_grpc

if __name__ == '__main__':
    with grpc.insecure_channel('localhost:50051') as channel:
        stub = rpc_pb2_grpc.PesoIdealStub(channel)

        while True:
            sexo = input("Sexo: [M/F] ")
            altura = input("Altura: ")

            response = stub.CalculaPesoIdeal(rpc_pb2.PesoIdealRequest(sexo=sexo, altura=float(altura)))
            print(f"Peso Ideal: {response.pesoIdeal}\n")
