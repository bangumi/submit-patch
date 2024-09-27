import enum
from datetime import datetime
from typing import Any, TypeVar
from uuid import UUID

import msgspec
from typing_extensions import Self


class PatchState(enum.IntEnum):
    Pending = 0
    Accept = 1
    Rejected = 2
    Outdated = 3


class PatchAction(enum.IntEnum):
    Unknown = 0
    Update = 1
    Create = 2


class SubjectType(enum.IntEnum):
    Book = 1
    Anime = 2
    Music = 3
    Game = 4
    Real = 6

    def __str__(self) -> str:
        return str(self.value)


T = TypeVar("T")


class PatchBase(msgspec.Struct, kw_only=True, frozen=True):
    id: UUID
    state: int
    from_user_id: int
    wiki_user_id: int
    reason: str
    created_at: datetime
    updated_at: datetime
    deleted_at: datetime | None
    reject_reason: str
    patch_desc: str  # extra description from user will not be included in commit message

    @classmethod
    def from_dict(cls, d: Any) -> Self:
        # will remove extra fields and do field level instance checking
        return msgspec.convert(d, cls)


class SubjectPatch(PatchBase, kw_only=True, frozen=True):
    subject_id: int
    subject_type: int

    original_name: str
    name: str | None

    original_infobox: str | None
    infobox: str | None

    original_summary: str | None
    summary: str | None

    nsfw: bool | None

    platform: int | None
    action: PatchAction


class EpisodePatch(PatchBase, kw_only=True, frozen=True):
    episode_id: int
    ep: int | None = None

    original_name: str
    name: str | None

    original_name_cn: str
    name_cn: str | None

    original_duration: str
    duration: str | None

    original_airdate: str
    airdate: str | None

    original_description: str
    description: str | None


class Wiki(msgspec.Struct, kw_only=True):
    name: str
    infobox: str
    summary: str
    nsfw: bool


class PatchType(str, enum.Enum):
    Subject = "subject"
    Episode = "episode"

    def __str__(self) -> str:
        return self.value
