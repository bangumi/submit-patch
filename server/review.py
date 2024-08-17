from dataclasses import dataclass
from datetime import datetime
from typing import Annotated, Any

import litestar
from litestar.enums import RequestEncodingType
from litestar.exceptions import NotAuthorizedException, NotFoundException
from litestar.params import Body
from litestar.response import Redirect, Template

from config import UTC
from server.auth import require_user_editor
from server.base import AuthorizedRequest, BadRequestException, http_client, pg
from server.model import Patch, React, State


@dataclass
class ReviewPatch:
    react: React


def __strip_none(d: dict[str, Any]) -> dict[str, Any]:
    return {key: value for key, value in d.items() if value is not None}


@litestar.post("/api/review-patch/{patch_id:str}", guards=[require_user_editor])
async def review_patch(
    patch_id: str,
    request: AuthorizedRequest,
    data: Annotated[ReviewPatch, Body(media_type=RequestEncodingType.URL_ENCODED)],
) -> Any:
    async with pg.acquire() as conn:
        async with conn.transaction():
            p = await pg.fetchrow(
                """select * from patch where id = $1 and deleted_at is NULL FOR UPDATE""", patch_id
            )
            if not p:
                raise NotFoundException()

            if p["state"] != State.Pending:
                raise BadRequestException("patch already reviewed")

            if data.react == React.Reject:
                await conn.execute(
                    """
                    update patch set
                        state = $1,
                        wiki_user_id = $2,
                        updated_at = $3
                    where id = $4 and deleted_at is NULL
                    """,
                    State.Rejected,
                    request.auth.user_id,
                    datetime.now(tz=UTC),
                    patch_id,
                )
                return Redirect("/")

    raise NotAuthorizedException("暂不支持")


@litestar.post("/api/review-patch/{patch_id:str}", guards=[require_user_editor])
async def review_patch2(
    patch_id: str,
    request: AuthorizedRequest,
    data: Annotated[ReviewPatch, Body(media_type=RequestEncodingType.URL_ENCODED)],
) -> Any:
    async with pg.acquire() as conn:
        async with conn.transaction():
            p = await pg.fetchrow(
                """select * from patch where id = $1 and deleted_at is NULL FOR UPDATE""", patch_id
            )
            if not p:
                raise BadRequestException("patch already reviewed")

            if data.react == React.Reject:
                await conn.execute(
                    """
                    update patch set
                        state = $1,
                        wiki_user_id = $2,
                        updated_at = $3
                    where id = $4 and deleted_at is NULL
                    """,
                    State.Rejected,
                    request.auth.user_id,
                    datetime.now(tz=UTC),
                    patch_id,
                )
                return Redirect("/")

            patch = Patch(**p)

            async with http_client.patch(
                f"https://next.bgm.tv/p1/wiki/subjects/{patch.subject_id}",
                json={
                    "commitMessage": patch.description,
                    "expectedRevision": __strip_none(
                        {
                            "infobox": patch.original_infobox,
                            "name": patch.original_name,
                            "summary": patch.original_summary,
                        }
                    ),
                    "subject": __strip_none(
                        {
                            "infobox": patch.infobox,
                            "name": patch.name,
                            "summary": patch.summary,
                            "nsfw": patch.nsfw,
                        }
                    ),
                },
            ) as res:
                print(await res.json())

            await conn.execute(
                """
                update patch set
                    state = $1,
                    wiki_user_id = $2,
                    updated_at = $3
                where id = $4 and deleted_at is NULL
                """,
                State.Accept,
                request.auth.user_id,
                datetime.now(tz=UTC),
                patch_id,
            )

            await pg.execute(
                "update patch set deleted_at = $1 where id = $2 ",
                datetime.now(tz=UTC),
                patch_id,
            )

            return Template("patch.html.jinja2", context={"patch": p, "auth": request.auth})
