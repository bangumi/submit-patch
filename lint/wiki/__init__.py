from __future__ import annotations

import dataclasses
from collections.abc import Generator


__all__ = (
    "ArrayItemWrappedError",
    "ArrayNoCloseError",
    "ExpectingNewFieldError",
    "ExpectingSignEqualError",
    "Field",
    "GlobalPrefixError",
    "GlobalSuffixError",
    "Item",
    "Wiki",
    "WikiSyntaxError",
    "parse",
    "render",
    "try_parse",
)


@dataclasses.dataclass(slots=True, frozen=True)
class Item:
    key: str = ""
    value: str = ""


@dataclasses.dataclass(slots=True, frozen=True)
class Field:
    key: str
    value: str | list[Item] | None = None


class Wiki:
    type: str | None
    fields: list[Field]

    __slots__ = "fields", "type"

    def __init__(self, type=None, fields: list[Field] | None = None):
        self.type = type

        if fields is None:
            self.fields = []
        else:
            self.fields = fields

    def non_zero(self) -> Wiki:
        wiki = Wiki(type=self.type)
        for f in self.fields:
            value = f.value

            if not value:
                continue

            if isinstance(value, str):
                if value:
                    wiki.fields.append(f)
                continue

            if isinstance(value, list):
                v = [x for x in value if x.key or x.value]
                if v:
                    wiki.fields.append(Field(f.key, value=v))
                continue

        return wiki


class WikiSyntaxError(ValueError):
    lino: int | None
    line: str | None
    message: str

    def __init__(self, lino: int | None = None, line: str | None = None, message: str = ""):
        self.line = line
        self.lino = lino
        self.message = message


class GlobalPrefixError(WikiSyntaxError):
    def __init__(self):
        super().__init__(message="missing prefix '{{Infobox' at the start")


class GlobalSuffixError(WikiSyntaxError):
    def __init__(self):
        super().__init__(message="missing '}}' at the end")


class ArrayNoCloseError(WikiSyntaxError):
    pass


class ArrayItemWrappedError(WikiSyntaxError):
    pass


class ExpectingNewFieldError(WikiSyntaxError):
    pass


class ExpectingSignEqualError(WikiSyntaxError):
    pass


def try_parse(s):
    """If failed to parse, return zero value"""
    try:
        return parse(s)
    except WikiSyntaxError:
        pass
    return Wiki()


prefix = "{{Infobox"
suffix = "}}"


def parse(s: str) -> Wiki:
    s = s.replace("\r\n", "\n")
    s, line_offset = _process_input(s)
    if not s:
        return Wiki()

    if not s.startswith(prefix):
        raise GlobalPrefixError()

    if not s.endswith(suffix):
        raise GlobalSuffixError

    w = Wiki(type=read_type(s))

    eol_count = s.count("\n")
    if eol_count <= 1:
        return w

    item_container: list[Item] = []

    # loop state
    in_array: bool = False
    current_key: str = ""

    for lino, line in enumerate(s.splitlines()):
        lino += line_offset

        # now handle line content
        line = _trim_space(line)
        if not line:
            continue

        if line[0] == "|":
            # new field
            if in_array:
                raise ArrayNoCloseError(lino, line)

            current_key = ""

            key, value = read_start_line(_trim_left_space(line[1:]))  # read "key = value"

            if not value:
                w.fields.append(Field(key=key))
                continue
            if value == "{":
                in_array = True
                current_key = key
                continue

            w.fields.append(Field(key=key, value=value))
            continue

        if in_array:
            if line == "}":  # close array
                in_array = False
                w.fields.append(Field(current_key, item_container))
                item_container = []
                continue

            # array item
            key, value = read_array_item(line)
            item_container.append(Item(key=key, value=value))

        # if not in_array:
        #     raise ErrExpectingNewField(lino, line)

    if in_array:
        # array should be close have read all contents
        raise ArrayNoCloseError(s.count("\n") + line_offset, s.splitlines()[-2])

    return w


def read_type(s):
    try:
        i = s.index("\n")
    except ValueError:
        i = s.index("}")  # {{Infobox Crt}}

    return _trim_space(s[len(prefix) : i])


def read_array_item(line):
    """Read whole line as an array item, spaces are trimmed.

    read_array_item("[简体中文名|鲁鲁修]") => "简体中文名", "鲁鲁修"
    read_array_item("[简体中文名|]") => "简体中文名", ""
    read_array_item("[鲁鲁修]") => "", "鲁鲁修"

    Raises:
        ArrayItemWrappedError: syntax error
    """
    if line[0] != "[" or line[len(line) - 1] != "]":
        raise ArrayItemWrappedError(-1, line)

    content = line[1 : len(line) - 1]

    try:
        i = content.index("|")
        return _trim_space(content[:i]), _trim_space(content[i + 1 :])
    except ValueError:
        return "", _trim_space(content)


def read_start_line(line: str):
    """Read line without leading '|' as key value pair, spaces are trimmed.

    read_start_line("播放日期 = 2017年4月16日") => 播放日期, 2017年4月16日
    read_start_line("播放日期 = ") => 播放日期, ""

    Raises:
        ExpectingSignEqualError: syntax error
    """
    try:
        i = line.index("=")
    except ValueError as e:
        raise ExpectingSignEqualError(0, line) from e

    return line[:i].strip(), line[i + 1 :].strip()


_space_str = " \t"


def _trim_space(s: str):
    return s.strip()


def _trim_left_space(s: str):
    return s.strip()


def _trim_right_space(s: str):
    return s.strip()


def _process_input(s: str) -> tuple[str, int]:
    offset = 2
    s = "\n".join(s.splitlines())

    for c in s:
        match c:
            case "\n":
                offset += 1
            case " ", "\t":
                continue
            case _:
                return s.strip(), offset

    return s.strip(), offset


def render(w: Wiki) -> str:
    return "\n".join(__render(w))


def __render(w: Wiki) -> Generator[str, None, None]:
    if w.type:
        yield "{{Infobox " + w.type
    else:
        yield "{{Infobox"

    for field in w.fields:
        if isinstance(field.value, str):
            yield f"| {field.key} = {field.value}"
        elif isinstance(field.value, list):
            yield f"| {field.key} = {{"
            yield from __render_items(field.value)
            yield "}"
        else:
            raise TypeError("type not support", type(field))

    yield "}}"


def __render_items(s: list[Item]):
    for item in s:
        if item.key:
            yield f"[{item.key}| {item.value}]"
        else:
            yield f"[{item.value}]"
