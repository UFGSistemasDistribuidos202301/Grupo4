import grpc
import rpc_pb2
import rpc_pb2_grpc

if __name__ == '__main__':
    with grpc.insecure_channel('localhost:50051') as channel:
        stub = rpc_pb2_grpc.AposentadoriaStub(channel)

        while True:
            idade = input("Idade: ")
            tempo_servico = input("Tempo de serviço: ")

            response = stub.PodeAposentar(rpc_pb2.PodeAposentarRequest(idade=int(idade), tempoServico=int(tempo_servico)))
            print(f"Pode aposentar: {response.podeAposentar}\n")
