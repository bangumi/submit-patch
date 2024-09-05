import enum
from dataclasses import dataclass
from datetime import datetime
from typing import Any, TypeVar
from uuid import UUID

from dacite import from_dict


class PatchState(enum.IntEnum):
    Pending = 0
    Accept = 1
    Rejected = 2
    Outdated = 3


T = TypeVar("T")


@dataclass(frozen=True, kw_only=True, slots=True)
class PatchBase:
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
    def from_dict(cls: type[T], d: Any) -> T:
        # will remove extra fields and do field level instance checking
        return from_dict(cls, d)


@dataclass(frozen=True, kw_only=True, slots=True)
class SubjectPatch(PatchBase):
    subject_id: int
    subject_type: int

    original_name: str
    name: str | None

    original_infobox: str | None
    infobox: str | None

    original_summary: str | None
    summary: str | None

    nsfw: bool | None


@dataclass(frozen=True, kw_only=True, slots=True)
class EpisodePatch(PatchBase):
    episode_id: int

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


@dataclass(frozen=True, slots=True, kw_only=True)
class Wiki:
    name: str
    infobox: str
    summary: str
    nsfw: bool


class PatchType(str, enum.Enum):
    Subject = "subject"
    Episode = "episode"

    def __str__(self) -> str:
        return self.value
