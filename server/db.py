import uuid
from datetime import datetime

import uuid6
from sqlalchemy.ext.asyncio import AsyncAttrs
from sqlalchemy.orm import DeclarativeBase, Mapped, mapped_column

from config import UTC


class Base(AsyncAttrs, DeclarativeBase):
    pass


class BasePatchMixin:
    id: Mapped[uuid.UUID] = mapped_column(primary_key=True, default=uuid6.uuid7)
    state: Mapped[int] = mapped_column()
    from_user_id: Mapped[int] = mapped_column()
    wiki_user_id: Mapped[int | None] = mapped_column()
    reason: Mapped[str] = mapped_column()
    created_at: Mapped[datetime] = mapped_column()
    updated_at: Mapped[datetime] = mapped_column(onupdate=lambda: datetime.now(tz=UTC))
    deleted_at: Mapped[datetime | None] = mapped_column()
    reject_reason: Mapped[str] = mapped_column()


class SubjectPatch(BasePatchMixin, Base):
    __tablename__ = "patch"

    subject_id: Mapped[int] = mapped_column()
    subject_type: Mapped[int] = mapped_column()
    original_name: Mapped[str] = mapped_column()
    name: Mapped[str | None] = mapped_column()
    original_infobox: Mapped[str | None] = mapped_column()
    infobox: Mapped[str | None] = mapped_column()
    original_summary: Mapped[str | None] = mapped_column()
    summary: Mapped[str | None] = mapped_column()
    nsfw: Mapped[bool | None] = mapped_column()


class EpisodePatch(BasePatchMixin, Base):
    __tablename__ = "episode_patch"

    episode_id: Mapped[int] = mapped_column()
    original_name: Mapped[str] = mapped_column()
    name: Mapped[str | None] = mapped_column()
    original_name_cn: Mapped[str] = mapped_column()
    name_cn: Mapped[str | None] = mapped_column()
    original_duration: Mapped[str] = mapped_column()
    duration: Mapped[str | None] = mapped_column()
    original_airdate: Mapped[str] = mapped_column()
    airdate: Mapped[str | None] = mapped_column()
    original_description: Mapped[str] = mapped_column()
    description: Mapped[str | None] = mapped_column()
