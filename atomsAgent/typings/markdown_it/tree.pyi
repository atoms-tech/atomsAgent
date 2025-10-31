from __future__ import annotations

from collections.abc import Generator, Sequence
from typing import Any, NamedTuple, TypeVar, overload

from markdown_it.token import Token

_NodeType = TypeVar("_NodeType", bound="SyntaxTreeNode")


class _NesterTokens(NamedTuple):
    opening: Token
    closing: Token


class SyntaxTreeNode:
    token: Token | None
    nester_tokens: _NesterTokens | None
    _parent: SyntaxTreeNode | None
    _children: list[SyntaxTreeNode]

    def __init__(
        self, tokens: Sequence[Token] = (), *, create_root: bool = True
    ) -> None: ...

    def __repr__(self) -> str: ...

    @overload
    def __getitem__(self: _NodeType, item: int) -> _NodeType: ...

    @overload
    def __getitem__(self: _NodeType, item: slice) -> list[_NodeType]: ...

    def __getitem__(self: _NodeType, item: int | slice) -> _NodeType | list[_NodeType]: ...

    def to_tokens(self: _NodeType) -> list[Token]: ...

    @property
    def children(self: _NodeType) -> list[_NodeType]: ...

    @children.setter
    def children(self: _NodeType, value: list[_NodeType]) -> None: ...

    @property
    def parent(self: _NodeType) -> _NodeType | None: ...

    @parent.setter
    def parent(self: _NodeType, value: _NodeType | None) -> None: ...

    @property
    def is_root(self) -> bool: ...

    @property
    def is_nested(self) -> bool: ...

    @property
    def siblings(self: _NodeType) -> Sequence[_NodeType]: ...

    @property
    def type(self) -> str: ...

    @property
    def next_sibling(self: _NodeType) -> _NodeType | None: ...

    @property
    def previous_sibling(self: _NodeType) -> _NodeType | None: ...

    def pretty(
        self, *, indent: int = 2, show_text: bool = False, _current: int = 0
    ) -> str: ...

    def walk(
        self: _NodeType, *, include_self: bool = True
    ) -> Generator[_NodeType, None, None]: ...

    @property
    def tag(self) -> str: ...

    @property
    def attrs(self) -> dict[str, str | int | float]: ...

    def attrGet(self, name: str) -> None | str | int | float: ...

    @property
    def map(self) -> tuple[int, int] | None: ...

    @property
    def level(self) -> int: ...

    @property
    def content(self) -> str: ...

    @property
    def markup(self) -> str: ...

    @property
    def info(self) -> str: ...

    @property
    def meta(self) -> dict[Any, Any]: ...

    @property
    def block(self) -> bool: ...

    @property
    def hidden(self) -> bool: ...
