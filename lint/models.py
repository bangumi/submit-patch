import difflib
import enum
from dataclasses import dataclass

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

    def patch(self) -> str:
        return "\n".join(difflib.unified_diff(self.origin.splitlines(), self.after.splitlines()))

    def __str__(self) -> str:
        return f"<Patch {self.category} {self.message}>"


@dataclass(frozen=True, slots=True, kw_only=True)
class SubjectWiki:
    id: int
    original: str
    type: SubjectType
    wiki: Wiki
