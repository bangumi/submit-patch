from __future__ import annotations

import re2

from server.errors import BadRequestException


__invisible_pattern = re2.compile(r"[^\t\r\n\p{L}\p{M}\p{N}\p{P}\p{S}\p{Z}]")


def check_invalid_input_str(*ss: str) -> None:
    for s in ss:
        for line in s.splitlines():
            if m := __invisible_pattern.search(line):
                raise BadRequestException(
                    "invalid character {!r} in line {!r}".format(m.group(0), line)
                )


def contains_invalid_input_str(*ss: str) -> str | None:
    for s in ss:
        if m := __invisible_pattern.search(s):
            return m.group(0)


def __repl(m: re2._Match[str]) -> str:
    return m.group(0).encode("unicode-escape").decode()


def escape_invisible(s: str) -> str:
    return __invisible_pattern.sub(__repl, s)
