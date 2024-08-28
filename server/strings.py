import io
import unicodedata

from server.base import BadRequestException


_invalid_category = {"Cf", "Mn", "Mc"}


def check_invalid_input_str(*s: str) -> None:
    for ss in s:
        for c in ss:
            v = unicodedata.category(c)
            if v in _invalid_category:
                raise BadRequestException("invalid character {!r}".format(c))


def invisible_escape(s: str) -> str:
    with io.StringIO() as f:
        for ss in s:
            for c in ss:
                v = unicodedata.category(c)
                if v in _invalid_category:
                    f.write(c.encode("unicode-escape").decode())
                else:
                    f.write(c)

        return f.getvalue()
