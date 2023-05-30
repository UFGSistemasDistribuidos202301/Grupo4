from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from typing import ClassVar as _ClassVar, Optional as _Optional

DESCRIPTOR: _descriptor.FileDescriptor

class PodeAposentarReply(_message.Message):
    __slots__ = ["podeAposentar"]
    PODEAPOSENTAR_FIELD_NUMBER: _ClassVar[int]
    podeAposentar: bool
    def __init__(self, podeAposentar: bool = ...) -> None: ...

class PodeAposentarRequest(_message.Message):
    __slots__ = ["idade", "tempoServico"]
    IDADE_FIELD_NUMBER: _ClassVar[int]
    TEMPOSERVICO_FIELD_NUMBER: _ClassVar[int]
    idade: int
    tempoServico: int
    def __init__(self, idade: _Optional[int] = ..., tempoServico: _Optional[int] = ...) -> None: ...
