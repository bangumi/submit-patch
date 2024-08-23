from collections.abc import Iterator

import litestar
from litestar.types import AnyCallable
from typing_extensions import TypeVar


T = TypeVar("T", bound=AnyCallable | litestar.Router)


class Router:
    """A helper class to collect handlers"""

    def __init__(self) -> None:
        self.__handler: list[AnyCallable | litestar.Router] = []

    def __call__(self, fn: T) -> T:
        self.__handler.append(fn)
        return fn

    def __iter__(self) -> Iterator[AnyCallable | litestar.Router]:
        yield from self.__handler
