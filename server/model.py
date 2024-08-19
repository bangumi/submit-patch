import enum
import time
from dataclasses import dataclass
from datetime import datetime


@dataclass(frozen=True, slots=True, kw_only=True)
class User:
    user_id: int
    group_id: int

    access_token: str
    refresh_token: str

    access_token_created_at: int  # unix time stamp
    access_token_expires_in: int  # seconds

    def is_access_token_fresh(self) -> bool:
        if not self.access_token:
            return False

        if self.access_token_created_at + self.access_token_expires_in <= time.time() + 120:
            return False

        return True

    @property
    def allow_edit(self) -> bool:
        return self.group_id in {2, 11}

    @property
    def allow_admin(self) -> bool:
        return self.group_id in {2}


class State(enum.IntEnum):
    Pending = 0
    Accept = 1
    Rejected = 2
    Outdated = 3


@dataclass(frozen=True, kw_only=True, slots=True)
class Patch:
    id: str
    subject_id: int
    state: int
    from_user_id: int
    wiki_user_id: int
    description: str
    name: str | None
    original_name: str | None
    infobox: str | None
    original_infobox: str | None
    summary: str | None
    original_summary: str | None
    nsfw: bool | None
    created_at: datetime
    updated_at: datetime
    deleted_at: datetime | None


@dataclass(frozen=True, slots=True, kw_only=True)
class Wiki:
    name: str
    infobox: str
    summary: str
    nsfw: bool
