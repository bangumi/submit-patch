from dataclasses import dataclass
from datetime import datetime
from typing import Annotated

import litestar
import orjson
from litestar.enums import RequestEncodingType
from litestar.exceptions import (
    HTTPException,
    NotAuthorizedException,
    NotFoundException,
    PermissionDeniedException,
    ValidationException,
)
from litestar.params import Body
from litestar.response import Redirect, Template

from config import TURNSTILE_SECRET_KEY, TURNSTILE_SITE_KEY, UTC
from server.base import BadRequestException, Request, http_client, pg
from server.model import Patch, Wiki


@litestar.get("/suggest")
async def suggest_ui(subject_id: int = 0) -> Template:
    if subject_id == 0:
        return Template("select-subject.html.jinja2")
    async with http_client.get(
        f"https://next.bgm.tv/p1/wiki/subjects/{subject_id}", allow_redirects=False
    ) as res:
        if res.status >= 300:
            raise NotFoundException()
        data = await res.json()
    return Template(
        "suggest.html.jinja2",
        context={"data": data, "subject_id": subject_id, "CAPTCHA_SITE_KEY": TURNSTILE_SITE_KEY},
    )


@dataclass(frozen=True, slots=True)
class CreateSuggestion:
    name: str
    infobox: str
    summary: str
    desc: str
    cf_turnstile_response: str
    # HTML form will only include checkbox when it's checked,
    # so any input is true, default value is false.
    nsfw: str | None = None


@litestar.post("/suggest")
async def suggest_api(
    subject_id: int,
    data: Annotated[CreateSuggestion, Body(media_type=RequestEncodingType.URL_ENCODED)],
    request: Request,
) -> Redirect:
    if not request.auth:
        raise PermissionDeniedException
    if request.auth.allow_edit:
        raise PermissionDeniedException

    if not data.desc:
        raise ValidationException("missing suggestion description")

    async with http_client.post(
        "https://challenges.cloudflare.com/turnstile/v0/siteverify",
        data={
            "secret": TURNSTILE_SECRET_KEY,
            "response": data.cf_turnstile_response,
        },
    ) as res:
        if res.status > 300:
            raise BadRequestException("验证码无效")
        captcha_data = orjson.loads(await res.read())
        if captcha_data.get("success") is not True:
            raise BadRequestException("验证码无效")

    async with http_client.get(f"https://next.bgm.tv/p1/wiki/subjects/{subject_id}") as res:
        res.raise_for_status()
        original_wiki = await res.json()

    original = Wiki(
        name=original_wiki["name"],
        infobox=original_wiki["infobox"],
        summary=original_wiki["summary"],
        nsfw=original_wiki["nsfw"],
    )

    name: str | None = None
    summary: str | None = None
    infobox: str | None = None

    original_name: str | None = None
    original_summary: str | None = None
    original_infobox: str | None = None

    nsfw: bool | None = None

    if original.name != data.name:
        name = data.name
        original_name = original.name

    if original.infobox != data.infobox:
        infobox = data.infobox
        original_infobox = original.infobox

    if original.summary != data.summary:
        summary = data.summary
        original_summary = original.summary

    if original.nsfw != data.nsfw is not None:  # true case
        nsfw = not original.nsfw

    if (name is None) and (summary is None) and (infobox is None) and (nsfw is None):
        raise HTTPException("no changes found", status_code=400)

    pk = await pg.fetchval(
        """
        insert into patch (subject_id, from_user_id, description, name, infobox, summary, nsfw,original_name,original_infobox,original_summary)
        VALUES ($1, $2, $3, $4, $5, $6, $7,$8,$9,$10)
        returning patch.id
    """,
        subject_id,
        request.auth.user_id,
        data.desc,
        name,
        infobox,
        summary,
        nsfw,
        original_name,
        original_infobox,
        original_summary,
    )

    return Redirect(f"/patch/{pk}")


@litestar.post("/api/delete-patch/{patch_id:str}")
async def delete_patch(patch_id: str, request: Request) -> Redirect:
    if not request.auth:
        raise NotAuthorizedException

    async with pg.acquire() as conn:
        async with conn.transaction():
            p = await conn.fetchrow(
                """select * from patch where id = $1 and deleted_at is NULL""", patch_id
            )
            if not p:
                raise NotFoundException()

            patch = Patch(**p)

            if patch.from_user_id != request.auth.user_id:
                raise NotAuthorizedException

            await conn.execute(
                "update patch set deleted_at = $1 where id = $2 ",
                datetime.now(tz=UTC),
                patch_id,
            )

            return Redirect("/")
