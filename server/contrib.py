import uuid
from dataclasses import dataclass
from datetime import datetime
from typing import Annotated, Any

import litestar
from litestar import Response
from litestar.enums import RequestEncodingType
from litestar.exceptions import (
    HTTPException,
    NotAuthorizedException,
    NotFoundException,
    ValidationException,
)
from litestar.params import Body
from litestar.response import Redirect, Template
from uuid6 import uuid7

from config import TURNSTILE_SECRET_KEY, UTC
from server.auth import require_user_login
from server.base import AuthorizedRequest, BadRequestException, Request, http_client, pg
from server.model import SubjectPatch
from server.router import Router


router = Router()


async def _validate_captcha(cf_turnstile_response: str) -> None:
    res = await http_client.post(
        "https://challenges.cloudflare.com/turnstile/v0/siteverify",
        data={
            "secret": TURNSTILE_SECRET_KEY,
            "response": cf_turnstile_response,
        },
    )
    if res.status_code > 300:
        raise BadRequestException("验证码无效")
    captcha_data = res.json()
    if captcha_data.get("success") is not True:
        raise BadRequestException("验证码无效")


@router
@litestar.get("/suggest")
async def suggest_ui(request: Request, subject_id: int = 0) -> Response[Any]:
    if subject_id == 0:
        return Template("select-subject.html.jinja2")

    if not request.auth:
        request.set_session({"backTo": request.url.path + f"?subject_id={subject_id}"})
        return Redirect("/login")

    res = await http_client.get(f"https://next.bgm.tv/p1/wiki/subjects/{subject_id}")
    if res.status_code >= 300:
        raise NotFoundException()
    data = res.json()
    return Template("suggest.html.jinja2", context={"data": data, "subject_id": subject_id})


@dataclass(frozen=True, slots=True, kw_only=True)
class CreateSuggestion:
    name: str
    infobox: str
    summary: str
    reason: str
    cf_turnstile_response: str
    # HTML form will only include checkbox when it's checked,
    # so any input is true, default value is false.
    nsfw: str | None = None


@router
@litestar.post("/suggest", guards=[require_user_login])
async def suggest_api(
    subject_id: int,
    data: Annotated[CreateSuggestion, Body(media_type=RequestEncodingType.URL_ENCODED)],
    request: AuthorizedRequest,
) -> Redirect:
    if not data.reason:
        raise ValidationException("missing suggestion description")

    await _validate_captcha(data.cf_turnstile_response)

    res = await http_client.get(f"https://next.bgm.tv/p1/wiki/subjects/{subject_id}")
    res.raise_for_status()
    original_wiki = res.json()

    original = {}

    changed = {}

    nsfw: bool | None = None

    for key in ["name", "infobox", "summary"]:
        before = original_wiki[key]
        after = getattr(data, key)
        if before != after:
            changed[key] = after
            original[key] = before

    if original_wiki["nsfw"] != (data.nsfw is not None):  # true case
        nsfw = not original_wiki["nsfw"]

    if (not changed) and (nsfw is None):
        raise HTTPException("no changes found", status_code=400)

    pk = uuid7()

    await pg.execute(
        """
        insert into patch (id, subject_id, from_user_id, reason, name, infobox, summary, nsfw,
                           original_name, original_infobox, original_summary, subject_type)
        VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
    """,
        pk,
        subject_id,
        request.auth.user_id,
        data.reason,
        changed.get("name"),
        changed.get("infobox"),
        changed.get("summary"),
        nsfw,
        original_wiki["name"],
        original.get("infobox"),
        original.get("summary"),
        original_wiki["typeID"],
    )

    return Redirect(f"/patch/{pk}")


@router
@litestar.post("/api/delete-patch/{patch_id:str}", guards=[require_user_login])
async def delete_patch(patch_id: str, request: AuthorizedRequest) -> Redirect:
    async with pg.acquire() as conn:
        async with conn.transaction():
            p = await conn.fetchrow(
                """select * from patch where id = $1 and deleted_at is NULL""", patch_id
            )
            if not p:
                raise NotFoundException()

            patch = SubjectPatch(**p)

            if patch.from_user_id != request.auth.user_id:
                raise NotAuthorizedException

            await conn.execute(
                "update patch set deleted_at = $1 where id = $2",
                datetime.now(tz=UTC),
                patch_id,
            )

            return Redirect("/")


@router
@litestar.get("/suggest-episode")
async def episode_suggest_ui(request: Request, episode_id: int = 0) -> Response[Any]:
    if episode_id == 0:
        return Template("episode/select.html.jinja2")

    if not request.auth:
        request.set_session({"backTo": request.url.path + f"?episode_id={episode_id}"})
        return Redirect("/login")

    res = await http_client.get(f"https://next.bgm.tv/p1/wiki/ep/{episode_id}")
    if res.status_code == 404:
        raise NotFoundException()

    res.raise_for_status()

    data = res.json()

    return Template("episode/suggest.html.jinja2", context={"data": data, "subject_id": episode_id})


@dataclass(frozen=True, slots=True, kw_only=True)
class CreateEpisodePatch:
    airdate: str
    name: str
    name_cn: str
    duration: str
    desc: str

    cf_turnstile_response: str
    reason: str


@router
@litestar.post("/suggest-episode", guards=[require_user_login])
async def creat_episode_patch(
    request: AuthorizedRequest,
    episode_id: int,
    data: Annotated[CreateEpisodePatch, Body(media_type=RequestEncodingType.URL_ENCODED)],
) -> Response[Any]:
    if not data.reason:
        raise ValidationException("missing suggestion description")

    await _validate_captcha(data.cf_turnstile_response)

    res = await http_client.get(f"https://next.bgm.tv/p1/wiki/ep/{episode_id}")
    if res.status_code == 404:
        raise NotFoundException()

    res.raise_for_status()

    org = res.json()
    original_wiki = {
        "airdate": org["date"],
        "name": org["name"],
        "name_cn": org["nameCN"],
        "duration": org["duration"],
        "desc": org["summary"],
    }

    keys = ["airdate", "name", "name_cn", "duration", "desc"]

    changed = {}

    for key in keys:
        if original_wiki[key] != getattr(data, key):
            changed[key] = getattr(data, key)

    if not changed:
        raise HTTPException("no changes found", status_code=400)

    pk = uuid7()

    await pg.execute(
        """
        insert into episode_patch (id, episode_id, from_user_id, reason, original_name, name,
            original_name_cn, name_cn, original_duration, duration,
            original_airdate, airdate, original_description, description)
        VALUES ($1, $2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14)
    """,
        pk,
        episode_id,
        request.auth.user_id,
        data.reason,
        original_wiki["name"],
        changed.get("name"),
        original_wiki["name_cn"],
        changed.get("name_cn"),
        original_wiki["duration"],
        changed.get("duration"),
        original_wiki["airdate"],
        changed.get("airdate"),
        original_wiki["desc"],
        changed.get("desc"),
    )

    return Redirect(f"/episode/{pk}")


@router
@litestar.post("/api/delete-episode/{patch_id:uuid}", guards=[require_user_login])
async def delete_episode_patch(patch_id: uuid.UUID, request: AuthorizedRequest) -> Redirect:
    async with pg.acquire() as conn:
        async with conn.transaction():
            p = await conn.fetchrow(
                """select from_user_id from episode_patch where id = $1 and deleted_at is NULL""",
                patch_id,
            )
            if not p:
                raise NotFoundException()

            if p["from_user_id"] != request.auth.user_id:
                raise NotAuthorizedException

            await conn.execute(
                "update episode_patch set deleted_at = $1 where id = $2",
                datetime.now(tz=UTC),
                patch_id,
            )

            return Redirect("/")
