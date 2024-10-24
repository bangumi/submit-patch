from __future__ import annotations

import enum
from dataclasses import dataclass
from datetime import datetime
from typing import TYPE_CHECKING, Annotated, Any
from uuid import UUID

import litestar
import msgspec.structs
from asyncpg import Record
from litestar import Controller, Response
from litestar.enums import RequestEncodingType
from litestar.exceptions import InternalServerException, NotAuthorizedException, NotFoundException
from litestar.params import Body
from litestar.response import Redirect
from sslog import logger

from server.auth import refresh_access_token, require_user_editor, require_user_login
from server.base import (
    CTX_REQUEST_ID,
    AuthorizedRequest,
    User,
    http_client,
    pg,
    session_key_back_to,
)
from server.config import UTC
from server.errors import BadRequestException
from server.model import EpisodePatch, PatchAction, PatchState, PatchType, SubjectPatch
from server.router import Router
from server.strings import check_invalid_input_str


if TYPE_CHECKING:
    from asyncpg.pool import PoolConnectionProxy

router = Router()


class React(str, enum.Enum):
    Accept = "accept"
    Reject = "reject"
    Comment = "comment"


@dataclass
class ReviewPatch:
    react: React
    text: str = ""


def _strip_none(d: dict[str, Any]) -> dict[str, Any]:
    return {key: value for key, value in d.items() if value is not None}


@router
class SubjectReviewController(Controller):
    @litestar.post(
        "/api/review-patch/{patch_id:str}",
        guards=[require_user_login],
        status_code=200,
    )
    async def review_patch(
        self,
        patch_id: UUID,
        request: AuthorizedRequest,
        data: Annotated[ReviewPatch, Body(media_type=RequestEncodingType.URL_ENCODED)],
    ) -> Response[Any]:
        logger.info("{} {} patch {}", request.auth.user_id, data.react.name, patch_id)
        if not request.auth.is_access_token_fresh():
            await refresh_access_token(request, back_to=f"/subject/{patch_id}")

        async with pg.acquire() as conn:
            async with conn.transaction():
                p = await pg.fetchrow(
                    """select * from view_subject_patch where id = $1 FOR UPDATE""",
                    patch_id,
                )
                if not p:
                    raise NotFoundException()

                patch = SubjectPatch.from_dict(p)

                msgspec.structs.replace(patch)

                if patch.state != PatchState.Pending:
                    raise BadRequestException("patch already reviewed")

                if data.react == React.Comment:
                    return await add_comment(
                        conn,
                        patch_id,
                        data.text.strip(),
                        request.auth.user_id,
                        patch_type=PatchType.Subject,
                    )

                require_user_editor(request)

                if data.react == React.Reject:
                    return await self.__reject_patch(patch, conn, request.auth, data.text.strip())

                if data.react == React.Accept:
                    if patch.action == PatchAction.Update:
                        return await self.__accept_patch_update(patch, conn, request)
                    if patch.action == PatchAction.Create:
                        return await self.__accept_patch_create(patch, conn, request)
                    raise NotImplementedError()

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
        return Redirect(f"/subject/{patch.id}")

    async def __accept_patch_create(
        self,
        patch: SubjectPatch,
        conn: PoolConnectionProxy[Record],
        request: AuthorizedRequest,
    ) -> Redirect:
        res = await http_client.post(
            "https://next.bgm.tv/p1/wiki/subjects",
            headers={
                "Authorization": f"Bearer {request.auth.access_token}",
                "cf-ray": CTX_REQUEST_ID.get(),
            },
            json={
                "name": patch.name,
                "type": patch.subject_type,
                "platform": patch.platform,
                "infobox": patch.infobox,
                "nsfw": patch.nsfw,
                "summary": patch.summary,
            },
        )
        if res.status_code >= 300:
            data: dict[str, Any] = res.json()
            err_code = data.get("code")
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
                    request.auth.user_id,
                    datetime.now(tz=UTC),
                    f"建议包含语法错误，已经自动拒绝: {data.get('message')}",
                    patch.id,
                )
                return Redirect(f"/subject/{patch.id}")

            if err_code == "TOKEN_INVALID":
                request.set_session({session_key_back_to: f"/subject/{patch.id}"})
                return Redirect("/login")

            logger.error(f"failed to apply patch {data!r}")
            raise InternalServerException()

        data = res.json()
        subject_id = data["subjectID"]

        await conn.execute(
            """
                    update subject_patch set
                        state = $1,
                        wiki_user_id = $2,
                        updated_at = $3,
                        subject_id = $4
                    where id = $5 and deleted_at is NULL
                    """,
            PatchState.Accept,
            request.auth.user_id,
            datetime.now(tz=UTC),
            subject_id,
            patch.id,
        )

        return Redirect(f"https://bgm.tv/subject/{subject_id}")

    async def __accept_patch_update(
        self,
        patch: SubjectPatch,
        conn: PoolConnectionProxy[Record],
        request: AuthorizedRequest,
    ) -> Redirect:
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
            headers={
                "Authorization": f"Bearer {request.auth.access_token}",
                "cf-ray": CTX_REQUEST_ID.get(),
            },
            json={
                "commitMessage": f"{patch.reason} [patch https://patch.bgm38.tv/subject/{patch.id}]",
                "expectedRevision": {
                    key: value
                    for key, value in [
                        ("infobox", patch.original_infobox),
                        ("name", patch.original_name),
                        ("summary", patch.original_summary),
                    ]
                    if key in subject
                },
                "subject": subject,
            },
        )
        if res.status_code >= 300:
            data: dict[str, Any] = res.json()
            err_code = data.get("code")
            if err_code == "WIKI_CHANGED":
                await conn.execute(
                    """
                    update view_subject_patch set
                        state = $1,
                        wiki_user_id = $2,
                        updated_at = $3,
                        reject_reason = $4
                    where id = $5
                    """,
                    PatchState.Outdated,
                    request.auth.user_id,
                    datetime.now(tz=UTC),
                    str(data.get("message", ""))[:255],
                    patch.id,
                )
                return Redirect(f"/subject/{patch.id}")

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
                    request.auth.user_id,
                    datetime.now(tz=UTC),
                    f"建议包含语法错误，已经自动拒绝: {data.get('message')}",
                    patch.id,
                )
                return Redirect(f"/subject/{patch.id}")

            if err_code == "TOKEN_INVALID":
                request.set_session({session_key_back_to: f"/subject/{patch.id}"})
                return Redirect("/login")

            logger.error(f"failed to apply patch {data!r}")
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
            request.auth.user_id,
            datetime.now(tz=UTC),
            patch.id,
        )

        next_pk = await conn.fetchval(
            """
            select id from view_subject_patch
            where state = $1 and created_at < $2
            order by created_at desc limit 1
            """,
            PatchState.Pending,
            patch.created_at,
        )

        if next_pk:
            return Redirect(f"/subject/{next_pk}")

        return Redirect("/?type=subject")


@router
class EpisodeReviewController(Controller):
    @litestar.post(
        "/api/review-episode/{patch_id:uuid}",
        guards=[require_user_editor],
        status_code=200,
    )
    async def review_episode_patch(
        self,
        patch_id: UUID,
        request: AuthorizedRequest,
        data: Annotated[ReviewPatch, Body(media_type=RequestEncodingType.URL_ENCODED)],
    ) -> Response[Any]:
        logger.info("{} {} patch {}", request.auth.user_id, data.react.name, patch_id)
        if not request.auth.is_access_token_fresh():
            await refresh_access_token(request, back_to=f"/episode/{patch_id}")

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
                        patch_id, conn, request.auth, data.text
                    )

                if data.react == React.Accept:
                    patch = EpisodePatch.from_dict(p)
                    return await self.__accept_episode_patch(patch, conn, request, request.auth)

        raise NotAuthorizedException("暂不支持")

    async def __reject_episode_patch(
        self, patch_id: UUID, conn: PoolConnectionProxy[Record], auth: User, reason: str
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
        return Redirect(f"/episode/{patch_id}")

    async def __accept_episode_patch(
        self,
        patch: EpisodePatch,
        conn: PoolConnectionProxy[Record],
        request: AuthorizedRequest,
        auth: User,
    ) -> Redirect:
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
            headers={
                "Authorization": f"Bearer {auth.access_token}",
                "cf-ray": CTX_REQUEST_ID.get(),
            },
            json={
                "commitMessage": f"{patch.reason} [patch https://patch.bgm38.tv/episode/{patch.id}]",
                "episode": episode,
            },
        )
        if res.status_code >= 300:
            data = res.json()
            err_code = data.get("code")
            if err_code == "TOKEN_INVALID":
                request.set_session({session_key_back_to: f"/episode/{patch.id}"})
                return Redirect("/login")

            logger.error(f"failed to apply patch {data!r}")
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

        next_pk = await conn.fetchval(
            """
            select id from view_episode_patch
            where state = $1 and created_at < $2
            order by created_at desc limit 1
            """,
            PatchState.Pending,
            patch.created_at,
        )

        if next_pk:
            return Redirect(f"/episode/{next_pk}")

        return Redirect("/?type=episode")


async def add_comment(
    conn: PoolConnectionProxy[Record],
    patch_id: UUID,
    text: str,
    from_user_id: int,
    patch_type: PatchType,
) -> Response[Any]:
    if not text:
        raise BadRequestException("请填写修改建议")

    check_invalid_input_str(text)

    await conn.execute(
        """
    insert into edit_suggestion (id, patch_id, patch_type, text, from_user)
    values (uuid_generate_v7(), $1, $2, $3, $4)
        """,
        patch_id,
        patch_type,
        text,
        from_user_id,
    )

    await conn.execute(
        f"""
        update {patch_type}_patch
            set comments_count = (
                select count(1)
                from edit_suggestion
                where patch_type = $1 and patch_id = $2
            )
        where id = $2
        """,
        patch_type,
        patch_id,
    )

    return Redirect(f"/{patch_type}/{patch_id}")
