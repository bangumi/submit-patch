from __future__ import annotations

import enum
from dataclasses import dataclass
from datetime import datetime
from typing import Annotated, Any

import litestar
from asyncpg import Record
from asyncpg.pool import PoolConnectionProxy
from litestar import Response
from litestar.enums import RequestEncodingType
from litestar.exceptions import InternalServerException, NotAuthorizedException, NotFoundException
from litestar.params import Body
from litestar.response import Redirect
from loguru import logger

from config import UTC
from server.auth import require_user_editor
from server.base import AuthorizedRequest, BadRequestException, User, http_client, pg
from server.model import Patch, PatchState


class React(str, enum.Enum):
    Accept = "accept"
    Reject = "reject"


@dataclass
class ReviewPatch:
    react: React
    reject_reason: str = ""


def __strip_none(d: dict[str, Any]) -> dict[str, Any]:
    return {key: value for key, value in d.items() if value is not None}


@litestar.post("/api/review-patch/{patch_id:str}", guards=[require_user_editor])
async def review_patch(
    patch_id: str,
    request: AuthorizedRequest,
    data: Annotated[ReviewPatch, Body(media_type=RequestEncodingType.URL_ENCODED)],
) -> Response[Any]:
    async with pg.acquire() as conn:
        async with conn.transaction():
            p = await pg.fetchrow(
                """select * from patch where id = $1 and deleted_at is NULL FOR UPDATE""", patch_id
            )
            if not p:
                raise NotFoundException()

            patch = Patch(**p)

            if patch.state != PatchState.Pending:
                raise BadRequestException("patch already reviewed")

            if data.react == React.Reject:
                return await __reject_patch(patch, conn, request.auth, data.reject_reason)

            if data.react == React.Accept:
                return await __accept_patch(patch, conn, request.auth)

    raise NotAuthorizedException("暂不支持")


async def __reject_patch(
    patch: Patch, conn: PoolConnectionProxy[Record], auth: User, reason: str
) -> Redirect:
    await conn.execute(
        """
        update patch set
            state = $1,
            wiki_user_id = $2,
            updated_at = $3,
            reject_reason = $4
        where id = $5 and deleted_at is NULL
        """,
        PatchState.Rejected,
        auth.user_id,
        datetime.now(tz=UTC),
        reason,
        patch.id,
    )
    return Redirect("/")


async def __accept_patch(patch: Patch, conn: PoolConnectionProxy[Record], auth: User) -> Redirect:
    if not auth.is_access_token_fresh():
        return Redirect("/login")

    subject = __strip_none(
        {
            "infobox": patch.infobox,
            "name": patch.name,
            "summary": patch.summary,
            "nsfw": patch.nsfw,
        }
    )

    res = await http_client.patch(
        f"https://next.bgm.tv/p1/wiki/subjects/{patch.subject_id}",
        headers={"Authorization": f"Bearer {auth.access_token}"},
        json={
            "commitMessage": f"{patch.description} [patch https://patch.bgm38.com/patch/{patch.id}]",
            "expectedRevision": {
                key: value
                for key, value in {
                    "infobox": patch.original_infobox,
                    "name": patch.original_name,
                    "summary": patch.original_summary,
                }.items()
                if key in subject
            },
            "subject": subject,
        },
    )
    if res.status_code >= 300:
        data = res.json()
        if data.get("code") == "SUBJECT_CHANGED":
            await conn.execute(
                """
                            update patch set
                                state = $1,
                                wiki_user_id = $2,
                                updated_at = $3
                            where id = $4 and deleted_at is NULL
                            """,
                PatchState.Outdated,
                auth.user_id,
                datetime.now(tz=UTC),
                patch.id,
            )
            return Redirect(f"/patch/{patch.id}")

        logger.error("failed to apply patch {!r}", data)
        raise InternalServerException()

    await conn.execute(
        """
                update patch set
                    state = $1,
                    wiki_user_id = $2,
                    updated_at = $3
                where id = $4 and deleted_at is NULL
                """,
        PatchState.Accept,
        auth.user_id,
        datetime.now(tz=UTC),
        patch.id,
    )
    return Redirect(f"/patch/{patch.id}")
