import regex

from server.base import BadRequestException


# stdlib re doesn't support `\p`
# re2 doesn't support negative lookahead
__invisible_pattern = regex.compile(r"(?![\t\r\n])(\p{Cf}|\p{Cc})")


def check_invalid_input_str(*ss: str) -> None:
    for s in ss:
        if m := __invisible_pattern.search(s):
            raise BadRequestException("invalid character {!r}".format(m.group(0)))


def __repl(m: regex.Match[str]) -> str:
    return m.group(1).encode("unicode-escape").decode()


def invisible_escape(s: str) -> str:
    return __invisible_pattern.sub(repl=__repl, string=s)
