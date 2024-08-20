import typing
from collections.abc import Callable
from datetime import datetime, timedelta
from typing import Any

import jinja2
from jinja2 import pass_context, select_autoescape
from jinja2.runtime import Context
from litestar import Request

from config import DEV, PROJECT_PATH, UTC


engine = jinja2.Environment(
    autoescape=select_autoescape(default=True),
    loader=jinja2.FileSystemLoader(PROJECT_PATH.joinpath("server", "templates")),
    auto_reload=DEV,
)

P = typing.ParamSpec("P")
T = typing.TypeVar("T")


@typing.overload
def add_filter(s: str) -> Callable[[Callable[P, T]], Callable[P, T]]: ...


@typing.overload
def add_filter(s: Callable[P, T]) -> Callable[P, T]: ...


def add_filter(s: str | Callable[P, T]) -> Any:
    def real_wrapper(name: str, fn: Callable[P, T]) -> Callable[P, T]:
        if name in engine.filters:
            raise ValueError(f"filter '{name}' already exists")
        engine.filters[name] = fn
        return fn

    if isinstance(s, str):
        return lambda fn: real_wrapper(s, fn)

    return real_wrapper(s.__name__, s)


@add_filter
@pass_context
def rel_time(ctx: Context, value: datetime) -> str:
    if not isinstance(value, datetime):
        raise TypeError("rel_time can be only called with datetime")

    req: Request[Any, Any, Any] | None = ctx.get("request")

    if req is None:
        return format_duration(datetime.now(tz=UTC) - value)

    return format_duration(req.state["now"] - value)


_Duration_Unit = [
    (60, "s"),
    (60, "m"),
    (24, "h"),
    (365, "d"),
    (1, "y"),
]


def format_duration(seconds: timedelta) -> str:
    s = " ago"

    dd = int(seconds.total_seconds())

    if dd <= 60:
        return "just now"

    for unit, unit_s in _Duration_Unit:
        dd, mod = divmod(dd, unit)
        if mod:
            s = f"{mod:.0f}{unit_s}" + s
        if dd == 0:
            break
    else:
        s = f"{int(dd)}{_Duration_Unit[-1][1]}" + s

    return s


@add_filter
def subject_type_readable(s: int) -> str:
    match s:
        case 1:
            return "书籍"
        case 2:
            return "动画"
        case 3:
            return "音乐"
        case 4:
            return "游戏"
        case 6:
            return "三次元"
        case _:
            return "Unknown"
