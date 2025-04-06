import html
import re
import typing
from collections.abc import Callable, Iterator
from datetime import datetime, timedelta, timezone
from typing import Any, cast
from urllib.parse import urlencode

import jinja2
from jinja2 import pass_context, select_autoescape
from jinja2.runtime import Context
from litestar import Request
from markupsafe import Markup
from multidict import MultiDict

from server.config import DEV, PROJECT_PATH, TURNSTILE_SITE_KEY, UTC


engine = jinja2.Environment(
    autoescape=select_autoescape(default=True),
    loader=jinja2.FileSystemLoader(PROJECT_PATH.joinpath("server", "templates")),
    auto_reload=DEV,
)


engine_globals: dict[str, Any] = engine.globals
engine_filters: dict[str, Any] = engine.filters

engine_globals["TURNSTILE_SITE_KEY"] = TURNSTILE_SITE_KEY
engine_globals["min"] = min
engine_globals["max"] = max

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
        engine_filters[name] = fn
        return fn

    if isinstance(s, str):

        def inner(fn: Callable[P, T]) -> Callable[P, T]:
            return real_wrapper(s, fn)

        return inner

    return real_wrapper(s.__name__, s)


def add_global_function(fn: Callable[P, T]) -> Callable[P, T]:
    name = fn.__name__
    if name in engine.globals:
        raise ValueError(f"filter '{name}' already exists")
    engine_globals[name] = fn
    return fn


@add_filter
@pass_context
def rel_time(ctx: Context, value: datetime) -> str:
    if not isinstance(value, datetime):
        raise TypeError("rel_time can be only called with datetime")

    req: Request[Any, Any, Any] | None = ctx.get("request")

    if req is None:
        return format_duration(datetime.now(tz=UTC) - value)

    return format_duration(req.state["now"] - value)


__duration_Unit = [
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

    if dd >= 3600:
        dd = dd - dd % 60

    for unit, unit_s in __duration_Unit:
        dd, mod = divmod(dd, unit)
        if mod:
            s = f"{mod:.0f}{unit_s}" + s
        if dd == 0:
            break
    else:
        s = f"{int(dd)}{__duration_Unit[-1][1]}" + s

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


local_tz = timezone(timedelta(hours=8), name="Asia/Shanghai")


@add_filter
@pass_context
def to_user_local_time(ctx: Context, dt: datetime) -> str:
    return dt.astimezone(ctx["request"].state.get("tz", local_tz)).strftime("%Y-%m-%d %H:%M:%S")


@add_global_function
@pass_context
def replace_url_query(ctx: Context, **kwargs: str) -> str:
    req: Request[None, None, Any] = ctx["request"]
    q: MultiDict[str] = req.url.query_params.copy()
    q.update(kwargs)
    return (
        req.url.path
        + "?"
        + urlencode(
            list(cast("Iterator[tuple[str, str]]", q.multi_items())),
        )
    )


# from https://stackoverflow.com/a/7160778/8062017
# https// http:// only
__url_pattern = re.compile(
    r"(https?://"  # http:// or https://
    r"[^/]+"  # netloc
    r"(?:/[^（），。() \r\n\s]*)?)",  # path#hash
    re.IGNORECASE,
)


def __repl_url(s: re.Match[str]) -> str:
    escaped = html.escape(s.group(0))
    return f'<a href="{escaped}" target="_blank">{escaped}</a>'


@add_filter
def auto_url(s: str) -> Markup:
    ss: list[str] = []

    end = 0
    for m in __url_pattern.finditer(s):
        ss.extend((html.escape(s[end : m.start()]), __repl_url(m)))
        end = m.end()

    if end != len(s):
        ss.append(html.escape(s[end:]))

    return Markup("".join(ss))
