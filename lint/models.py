import difflib
import enum

import pydantic

from lint.wiki import Wiki


class SubjectType(str, enum.Enum):
    anime = "anime"
    book = "book"
    music = "music"
    game = "game"
    real = "real"


class Patch:
    def __init__(
        self,
        original: str,
        after: str,
        message: str,
        category: str,
        simple: bool = False,
    ):
        self.after = after
        self.origin = original
        self.message = message
        self.category = category
        self.simple = simple

    def patch(self):
        return "\n".join(difflib.unified_diff(self.origin.splitlines(), self.after.splitlines()))

    def __str__(self):
        return f"<Patch {self.category} {self.message}>"


class SubjectWiki(pydantic.BaseModel):
    id: int
    original: str
    type: SubjectType
    wiki: Wiki
