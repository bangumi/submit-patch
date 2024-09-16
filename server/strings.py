import regex

from server.errors import BadRequestException


# stdlib re doesn't support `\p`
# re2 doesn't support negative lookahead
__invisible_pattern = regex.compile(r"(?![\t\r\n])(\p{Cf}|\p{Cc}|\p{Co})")


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


def __repl(m: regex.Match[str]) -> str:
    return m.group(0).encode("unicode-escape").decode()


def escape_invisible(s: str) -> str:
    return __invisible_pattern.sub(repl=__repl, string=s)
