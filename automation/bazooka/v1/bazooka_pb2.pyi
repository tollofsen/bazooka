from google.protobuf.internal import containers as _containers
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class OutputKind(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    OUTPUT_KIND_UNSPECIFIED: _ClassVar[OutputKind]
    OUTPUT_KIND_GAME: _ClassVar[OutputKind]
    OUTPUT_KIND_COMMAND: _ClassVar[OutputKind]
    OUTPUT_KIND_SYSTEM: _ClassVar[OutputKind]
OUTPUT_KIND_UNSPECIFIED: OutputKind
OUTPUT_KIND_GAME: OutputKind
OUTPUT_KIND_COMMAND: OutputKind
OUTPUT_KIND_SYSTEM: OutputKind

class ConnectRequest(_message.Message):
    __slots__ = ("host", "port", "session_id")
    HOST_FIELD_NUMBER: _ClassVar[int]
    PORT_FIELD_NUMBER: _ClassVar[int]
    SESSION_ID_FIELD_NUMBER: _ClassVar[int]
    host: str
    port: int
    session_id: str
    def __init__(self, host: _Optional[str] = ..., port: _Optional[int] = ..., session_id: _Optional[str] = ...) -> None: ...

class ConnectResponse(_message.Message):
    __slots__ = ("session_id",)
    SESSION_ID_FIELD_NUMBER: _ClassVar[int]
    session_id: str
    def __init__(self, session_id: _Optional[str] = ...) -> None: ...

class DisconnectRequest(_message.Message):
    __slots__ = ("session_id",)
    SESSION_ID_FIELD_NUMBER: _ClassVar[int]
    session_id: str
    def __init__(self, session_id: _Optional[str] = ...) -> None: ...

class DisconnectResponse(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class SendCommandRequest(_message.Message):
    __slots__ = ("session_id", "command", "source")
    SESSION_ID_FIELD_NUMBER: _ClassVar[int]
    COMMAND_FIELD_NUMBER: _ClassVar[int]
    SOURCE_FIELD_NUMBER: _ClassVar[int]
    session_id: str
    command: str
    source: str
    def __init__(self, session_id: _Optional[str] = ..., command: _Optional[str] = ..., source: _Optional[str] = ...) -> None: ...

class SendCommandResponse(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class StreamOutputRequest(_message.Message):
    __slots__ = ("session_id", "replay_lines")
    SESSION_ID_FIELD_NUMBER: _ClassVar[int]
    REPLAY_LINES_FIELD_NUMBER: _ClassVar[int]
    session_id: str
    replay_lines: int
    def __init__(self, session_id: _Optional[str] = ..., replay_lines: _Optional[int] = ...) -> None: ...

class Output(_message.Message):
    __slots__ = ("session_id", "kind", "text", "timestamp_unix_ms")
    SESSION_ID_FIELD_NUMBER: _ClassVar[int]
    KIND_FIELD_NUMBER: _ClassVar[int]
    TEXT_FIELD_NUMBER: _ClassVar[int]
    TIMESTAMP_UNIX_MS_FIELD_NUMBER: _ClassVar[int]
    session_id: str
    kind: OutputKind
    text: str
    timestamp_unix_ms: int
    def __init__(self, session_id: _Optional[str] = ..., kind: _Optional[_Union[OutputKind, str]] = ..., text: _Optional[str] = ..., timestamp_unix_ms: _Optional[int] = ...) -> None: ...

class ListSessionsRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class SessionInfo(_message.Message):
    __slots__ = ("session_id", "host", "port", "connected", "started_unix_ms")
    SESSION_ID_FIELD_NUMBER: _ClassVar[int]
    HOST_FIELD_NUMBER: _ClassVar[int]
    PORT_FIELD_NUMBER: _ClassVar[int]
    CONNECTED_FIELD_NUMBER: _ClassVar[int]
    STARTED_UNIX_MS_FIELD_NUMBER: _ClassVar[int]
    session_id: str
    host: str
    port: int
    connected: bool
    started_unix_ms: int
    def __init__(self, session_id: _Optional[str] = ..., host: _Optional[str] = ..., port: _Optional[int] = ..., connected: _Optional[bool] = ..., started_unix_ms: _Optional[int] = ...) -> None: ...

class ListSessionsResponse(_message.Message):
    __slots__ = ("sessions",)
    SESSIONS_FIELD_NUMBER: _ClassVar[int]
    sessions: _containers.RepeatedCompositeFieldContainer[SessionInfo]
    def __init__(self, sessions: _Optional[_Iterable[_Union[SessionInfo, _Mapping]]] = ...) -> None: ...
