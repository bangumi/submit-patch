import unicodedata

from server.base import BadRequestException


def check_invalid_input_str(*s: str):
    for ss in s:
        for c in ss:
            v = unicodedata.category(c)
            if v == "Cf":  # Format
                raise BadRequestException("invalid character {!r}".format(c))
