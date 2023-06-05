from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from typing import ClassVar as _ClassVar, Optional as _Optional

DESCRIPTOR: _descriptor.FileDescriptor

class PesoIdealReply(_message.Message):
    __slots__ = ["pesoIdeal"]
    PESOIDEAL_FIELD_NUMBER: _ClassVar[int]
    pesoIdeal: float
    def __init__(self, pesoIdeal: _Optional[float] = ...) -> None: ...

class PesoIdealRequest(_message.Message):
    __slots__ = ["altura", "sexo"]
    ALTURA_FIELD_NUMBER: _ClassVar[int]
    SEXO_FIELD_NUMBER: _ClassVar[int]
    altura: float
    sexo: str
    def __init__(self, sexo: _Optional[str] = ..., altura: _Optional[float] = ...) -> None: ...
