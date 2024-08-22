import enum
from dataclasses import dataclass
from datetime import datetime


class PatchState(enum.IntEnum):
    Pending = 0
    Accept = 1
    Rejected = 2
    Outdated = 3


@dataclass(frozen=True, kw_only=True, slots=True)
class Patch:
    id: str
    subject_id: int
    subject_type: int
    state: int
    from_user_id: int
    wiki_user_id: int
    description: str
    name: str | None
    original_name: str
    infobox: str | None
    original_infobox: str | None
    summary: str | None
    original_summary: str | None
    nsfw: bool | None
    created_at: datetime
    updated_at: datetime
    deleted_at: datetime | None
    reject_reason: str


@dataclass(frozen=True, kw_only=True, slots=True)
class EpisodePatch:
    id: str
    episode_id: int
    state: int
    from_user_id: int
    wiki_user_id: int
    reason: str
    original_name: str
    name: str
    original_name_cn: str
    name_cn: str
    original_duration: str
    duration: str | None
    original_airdate: str
    airdate: str | None
    original_description: str
    description: str | None
    created_at: str
    updated_at: str
    deleted_at: datetime | None
    reject_reason: str


@dataclass(frozen=True, slots=True, kw_only=True)
class Wiki:
    name: str
    infobox: str
    summary: str
    nsfw: bool
