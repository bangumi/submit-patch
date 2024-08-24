from __future__ import annotations

import enum
import uuid
from dataclasses import dataclass
from datetime import datetime
from typing import Annotated, Any

import litestar
import pydash
from asyncpg import Record
from asyncpg.pool import PoolConnectionProxy
from dacite import from_dict
from litestar import Controller, Response, params
from litestar.enums import RequestEncodingType
from litestar.exceptions import InternalServerException, NotAuthorizedException, NotFoundException
from litestar.params import Body
from litestar.response import Redirect
from loguru import logger
from uuid_utils import uuid7

from config import UTC
from server.auth import require_user_editor
from server.base import AuthorizedRequest, BadRequestException, User, http_client, pg
from server.model import EpisodePatch, PatchState, PatchType, SubjectPatch
from server.router import Router


router = Router()


class React(str, enum.Enum):
    Accept = "accept"
    Reject = "reject"


@dataclass
class ReviewPatch:
    react: React
    reject_reason: str = ""


def _strip_none(d: dict[str, Any]) -> dict[str, Any]:
    return {key: value for key, value in d.items() if value is not None}


@router
class SubjectReviewController(Controller):
    @litestar.post("/api/review-patch/{patch_id:str}", guards=[require_user_editor])
    async def review_patch(
        self,
        patch_id: str,
        request: AuthorizedRequest,
        data: Annotated[ReviewPatch, Body(media_type=RequestEncodingType.URL_ENCODED)],
    ) -> Response[Any]:
        async with pg.acquire() as conn:
            async with conn.transaction():
                p = await pg.fetchrow(
                    """select * from view_subject_patch where id = $1 FOR UPDATE""",
                    patch_id,
                )
                if not p:
                    raise NotFoundException()

                patch = SubjectPatch(**p)

                if patch.state != PatchState.Pending:
                    raise BadRequestException("patch already reviewed")

                if data.react == React.Reject:
                    return await self.__reject_patch(patch, conn, request.auth, data.reject_reason)

                if data.react == React.Accept:
                    return await self.__accept_patch(patch, conn, request.auth)

        raise NotAuthorizedException("暂不支持")

    async def __reject_patch(
        self,
        patch: SubjectPatch,
        conn: PoolConnectionProxy[Record],
        auth: User,
        reason: str,
    ) -> Redirect:
        await conn.execute(
            """
            update view_subject_patch set
                state = $1,
                wiki_user_id = $2,
                updated_at = $3,
                reject_reason = $4
            where id = $5
            """,
            PatchState.Rejected,
            auth.user_id,
            datetime.now(tz=UTC),
            reason,
            patch.id,
        )
        return Redirect("/")

    async def __accept_patch(
        self,
        patch: SubjectPatch,
        conn: PoolConnectionProxy[Record],
        auth: User,
    ) -> Redirect:
        if not auth.is_access_token_fresh():
            return Redirect("/login")

        subject = _strip_none(
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
                "commitMessage": f"{patch.reason} [patch https://patch.bgm38.tv/patch/{patch.id}]",
                "expectedRevision": pydash.pick(
                    {
                        "infobox": patch.original_infobox,
                        "name": patch.original_name,
                        "summary": patch.original_summary,
                    },
                    *subject.keys(),
                ),
                "subject": subject,
            },
        )
        if res.status_code >= 300:
            data: dict[str, Any] = res.json()
            err_code = data.get("code")
            if err_code == "SUBJECT_CHANGED":
                await conn.execute(
                    """
                    update view_subject_patch set
                        state = $1,
                        wiki_user_id = $2,
                        updated_at = $3
                    where id = $4
                    """,
                    PatchState.Outdated,
                    auth.user_id,
                    datetime.now(tz=UTC),
                    patch.id,
                )
                return Redirect("/")

            if err_code == "INVALID_SYNTAX_ERROR":
                await conn.execute(
                    """
                    update view_subject_patch set
                        state = $1,
                        wiki_user_id = $2,
                        updated_at = $3,
                        reject_reason = $4
                    where id = $5
                    """,
                    PatchState.Rejected,
                    auth.user_id,
                    datetime.now(tz=UTC),
                    f"建议包含语法错误，已经自动拒绝: {data.get('message')}",
                    patch.id,
                )
                raise BadRequestException("建议包含语法错误，已经自动拒绝")

            logger.error("failed to apply patch {!r}", data)
            raise InternalServerException()

        await conn.execute(
            """
                    update subject_patch set
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
        return Redirect("/")


@router
class EpisodeReviewController(Controller):
    @litestar.post("/api/review-episode/{patch_id:uuid}", guards=[require_user_editor])
    async def review_episode_patch(
        self,
        patch_id: uuid.UUID,
        request: AuthorizedRequest,
        data: Annotated[ReviewPatch, Body(media_type=RequestEncodingType.URL_ENCODED)],
    ) -> Response[Any]:
        async with pg.acquire() as conn:
            async with conn.transaction():
                p = await pg.fetchrow(
                    """select * from view_episode_patch where id = $1 FOR UPDATE""",
                    patch_id,
                )
                if not p:
                    raise NotFoundException()

                if p["state"] != PatchState.Pending:
                    raise BadRequestException("patch already reviewed")

                if data.react == React.Reject:
                    return await self.__reject_episode_patch(
                        patch_id, conn, request.auth, data.reject_reason
                    )

                if data.react == React.Accept:
                    patch = from_dict(EpisodePatch, p)  # type: ignore
                    return await self.__accept_episode_patch(patch, conn, request.auth)

        raise NotAuthorizedException("暂不支持")

    async def __reject_episode_patch(
        self, patch_id: uuid.UUID, conn: PoolConnectionProxy[Record], auth: User, reason: str
    ) -> Redirect:
        await conn.execute(
            """
            update view_episode_patch set
                state = $1,
                wiki_user_id = $2,
                updated_at = $3,
                reject_reason = $4
            where id = $5
            """,
            PatchState.Rejected,
            auth.user_id,
            datetime.now(tz=UTC),
            reason,
            patch_id,
        )
        return Redirect("/?type=episode")

    async def __accept_episode_patch(
        self, patch: EpisodePatch, conn: PoolConnectionProxy[Record], auth: User
    ) -> Redirect:
        if not auth.is_access_token_fresh():
            return Redirect("/login")

        episode = _strip_none(
            {
                "nameCN": patch.name_cn,
                "name": patch.name,
                "summary": patch.description,
                "duration": patch.duration,
                "date": patch.airdate,
            }
        )

        res = await http_client.patch(
            f"https://next.bgm.tv/p1/wiki/ep/{patch.episode_id}",
            headers={"Authorization": f"Bearer {auth.access_token}"},
            json={
                "commitMessage": f"{patch.reason} [patch https://patch.bgm38.tv/patch/{patch.id}]",
                "episode": episode,
            },
        )
        if res.status_code >= 300:
            data = res.json()
            logger.error("failed to apply patch {!r}", data)
            raise InternalServerException()

        await conn.execute(
            """
            update view_episode_patch set
                state = $1,
                wiki_user_id = $2,
                updated_at = $3
            where id = $4
            """,
            PatchState.Accept,
            auth.user_id,
            datetime.now(tz=UTC),
            patch.id,
        )
        return Redirect("/?type=episode")


@dataclass(slots=True, frozen=True)
class CommentOnPatch:
    text: str = ""


@router
class CommentReviewController(Controller):
    @litestar.post("/api/add-suggestion/{patch_id:uuid}", guards=[require_user_editor])
    async def handler(
        self,
        request: AuthorizedRequest,
        patch_id: uuid.UUID,
        data: Annotated[CommentOnPatch, Body(media_type=RequestEncodingType.URL_ENCODED)],
        patch_type: Annotated[PatchType, params.Parameter(query="type")] = PatchType.Subject,
    ) -> Response[Any]:
        if patch_type == PatchType.Subject:
            p = await pg.fetchval(
                "select id from view_subject_patch where id = $1 AND state = $2",
                patch_id,
                PatchState.Pending,
            )
        else:
            p = await pg.fetchval(
                "select id from view_episode_patch where id = $1 AND state = $2",
                patch_id,
                PatchState.Pending,
            )
        if not p:
            raise NotFoundException("patch not found")

        await pg.execute(
            """
        insert into edit_suggestion (id, patch_id, patch_type, text, from_user)
        values ($1, $2, $3, $4, $5)
            """,
            uuid7(),
            patch_id,
            patch_type,
            data.text,
            request.auth.user_id,
        )

        if patch_type == PatchType.Subject:
            return Redirect(f"/patch/{patch_id}")

        return Redirect(f"/episode/{patch_id}")
