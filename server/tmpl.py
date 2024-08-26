import html
import re
import typing
from collections.abc import Callable
from datetime import datetime, timedelta, timezone
from typing import Any
from urllib.parse import urlencode

import jinja2
from jinja2 import pass_context, select_autoescape
from jinja2.runtime import Context
from litestar import Request
from markupsafe import Markup
from typing_extensions import Never

from config import DEV, PROJECT_PATH, TURNSTILE_SITE_KEY, UTC


engine = jinja2.Environment(
    autoescape=select_autoescape(default=True),
    loader=jinja2.FileSystemLoader(PROJECT_PATH.joinpath("server", "templates")),
    auto_reload=DEV,
)

engine.globals["TURNSTILE_SITE_KEY"] = TURNSTILE_SITE_KEY
engine.globals["min"] = min
engine.globals["max"] = max


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


def add_global_function(fn: Callable[P, T]) -> Callable[P, T]:
    name = fn.__name__
    if name in engine.globals:
        raise ValueError(f"filter '{name}' already exists")
    engine.globals[name] = fn
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
def to_user_local_time(dt: datetime) -> str:
    return str(dt.replace(microsecond=0).astimezone(local_tz))


@add_global_function
@pass_context
def replace_url_query(ctx: Context, **kwargs: Any) -> str:
    req: Request[Never, Never, Never] = ctx["request"]
    q = req.url.query_params.copy()
    for key, value in kwargs.items():
        q[key] = str(value)
    return req.url.path + "?" + urlencode(q)


# from https://stackoverflow.com/a/7160778/8062017
# https// http:// only
is_url_pattern = re.compile(
    r"^https?://"  # http:// or https://
    r"(?:(?:[A-Z0-9](?:[A-Z0-9-]{0,61}[A-Z0-9])?\.)+(?:[A-Z]{2,6}\.?|[A-Z0-9-]{2,}\.?)|"  # domain...
    r"localhost|"  # localhost...d
    r"\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3})"  # ...or ip
    r"(?::\d+)?"  # optional port
    r"(?:/?|[/?]\S+)$",
    re.IGNORECASE,
)


def __render_maybe_url(s: str) -> str:
    if is_url_pattern.match(s):
        escaped = html.escape(s)
        return f'<a href="{escaped}" target="_blank">{escaped}</a>'
    return s


__split_pattern = re.compile(r"([ （）()])")


@add_filter
def auto_url(s: str) -> Markup:
    lines = s.splitlines()

    ss = []

    for line in lines:
        ss.append("".join(__render_maybe_url(x) for x in __split_pattern.split(line)))

    return Markup("<br>".join(ss))
