from datetime import datetime, timedelta

import jinja2
from jinja2 import pass_context, select_autoescape
from jinja2.runtime import Context
from litestar import Request

from config import DEV, PROJECT_PATH, UTC


engine = jinja2.Environment(
    autoescape=select_autoescape(default=True),
    loader=jinja2.FileSystemLoader(PROJECT_PATH.joinpath("server", "templates")),
    cache_size=int(DEV) * 400,
    auto_reload=not DEV,
)


@pass_context
def rel_time(ctx: Context, value: datetime):
    if not isinstance(value, datetime):
        raise TypeError("rel_time can be only called with datetime")

    req: Request | None = ctx.get("request")

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


engine.filters["rel_time"] = rel_time
