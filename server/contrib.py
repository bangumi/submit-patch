from dataclasses import dataclass
from datetime import datetime
from typing import Annotated, Any
from uuid import UUID

import litestar
from litestar import Response
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
from uuid_utils.compat import uuid7

from server.auth import require_user_login
from server.base import (
    AuthorizedRequest,
    BadRequestException,
    QueueItem,
    Request,
    http_client,
    patch_keys,
    pg,
    session_key_back_to,
    subject_infobox_queue,
)
from server.config import TURNSTILE_SECRET_KEY, UTC
from server.model import PatchState, SubjectPatch
from server.router import Router
from server.strings import check_invalid_input_str, contains_invalid_input_str


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
@litestar.get(["/suggest", "/suggest-subject"])
async def suggest_ui(request: Request, subject_id: int = 0) -> Response[Any]:
    if subject_id == 0:
        return Template("select-subject.html.jinja2")

    if not request.auth:
        request.set_session({session_key_back_to: f"/suggest-subject?subject_id={subject_id}"})
        return Redirect("/login")

    res = await http_client.get(f"https://next.bgm.tv/p1/wiki/subjects/{subject_id}")
    if res.status_code >= 300:
        raise NotFoundException()
    data = res.json()
    return Template("suggest.html.jinja2", context={"data": data, "subject_id": subject_id})


@dataclass(frozen=True, slots=True, kw_only=True)
class CreateSubjectPatch:
    name: str
    infobox: str
    summary: str
    reason: str
    patch_desc: str
    cf_turnstile_response: str
    # HTML form will only include checkbox when it's checked,
    # so any input is true, default value is false.
    nsfw: str | None = None


@router
@litestar.post(
    "/suggest-subject",
    guards=[require_user_login],
    status_code=200,
)
async def suggest_api(
    subject_id: int,
    data: Annotated[CreateSubjectPatch, Body(media_type=RequestEncodingType.URL_ENCODED)],
    request: AuthorizedRequest,
) -> Redirect:
    if not data.reason:
        raise ValidationException("missing suggestion description")

    check_invalid_input_str(data.reason, data.patch_desc)

    if not request.auth.allow_bypass_captcha():
        await _validate_captcha(data.cf_turnstile_response)

    res = await http_client.get(f"https://next.bgm.tv/p1/wiki/subjects/{subject_id}")
    res.raise_for_status()
    original_wiki = res.json()

    original = {}

    changed = {}

    for key in ["name", "infobox", "summary"]:
        before = original_wiki[key]
        after = getattr(data, key)
        if before != after:
            changed[key] = after
            original[key] = before

    nsfw: bool | None = None
    nsfw_input = data.nsfw is not None
    if original_wiki["nsfw"] != nsfw_input:  # true case
        nsfw = nsfw_input

    if (not changed) and (nsfw is None):
        raise HTTPException("no changes found", status_code=400)

    for key in ["name", "infobox", "summary"]:
        if key in changed:
            check_invalid_input_str(changed[key])

    pk = uuid7()

    await pg.execute(
        """
        insert into subject_patch (id, subject_id, from_user_id, reason, name, infobox, summary, nsfw,
                           original_name, original_infobox, original_summary, subject_type, patch_desc)
        VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
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
        data.patch_desc,
    )

    if "infobox" in changed:
        await subject_infobox_queue.put(QueueItem(infobox=changed["infobox"], patch_id=pk))

    return Redirect(f"/subject/{pk}")


@router
@litestar.post(
    "/api/delete-subject/{patch_id:str}",
    guards=[require_user_login],
    status_code=200,
)
async def delete_patch(patch_id: str, request: AuthorizedRequest) -> Redirect:
    async with pg.acquire() as conn:
        async with conn.transaction():
            p = await conn.fetchrow(
                """select * from subject_patch where id = $1 and deleted_at is NULL""", patch_id
            )
            if not p:
                raise NotFoundException()

            patch = SubjectPatch.from_dict(p)

            if patch.from_user_id != request.auth.user_id:
                raise NotAuthorizedException("you are not owner of this patch")

            if patch.state != PatchState.Pending:
                raise NotAuthorizedException("patch 已经被审核")

            await conn.execute(
                "update subject_patch set deleted_at = $1 where id = $2",
                datetime.now(tz=UTC),
                patch_id,
            )

            return Redirect("/")


@router
@litestar.get("/edit/subject/{patch_id:uuid}", guards=[require_user_login])
async def _(request: AuthorizedRequest, patch_id: UUID) -> Response[Any]:
    p = await pg.fetchrow(
        "select * from view_subject_patch where id = $1",
        patch_id,
    )
    if not p:
        raise NotFoundException()

    if p["from_user_id"] != request.auth.user_id:
        raise PermissionDeniedException("you are not owner of this patch")

    if p["state"] != PatchState.Pending:
        raise BadRequestException("patch 已经被审核")

    res = await http_client.get(f"https://next.bgm.tv/p1/wiki/subjects/{p['subject_id']}")
    res.raise_for_status()
    wiki = res.json()

    return Template(
        "suggest.html.jinja2",
        context={
            "data": wiki | {key: value for key, value in p.items() if value is not None},
            "patch_id": patch_id,
        },
    )


@dataclass(frozen=True, slots=True, kw_only=True)
class EditSubjectPatch:
    name: str | None = None
    infobox: str | None = None
    summary: str | None = None
    reason: str
    cf_turnstile_response: str
    # HTML form will only include checkbox when it's checked,
    # so any input is true, default value is false.
    nsfw: str | None = None


@router
@litestar.post(
    "/edit/subject/{patch_id:uuid}",
    guards=[require_user_login],
    status_code=200,
)
async def _(
    request: AuthorizedRequest,
    patch_id: UUID,
    data: Annotated[EditSubjectPatch, Body(media_type=RequestEncodingType.URL_ENCODED)],
) -> Response[Any]:
    await _validate_captcha(data.cf_turnstile_response)

    check_invalid_input_str(
        *[x for x in [data.name, data.infobox, data.summary, data.reason] if x is not None]
    )

    async with pg.acquire() as conn:
        async with conn.transaction():
            p = await conn.fetchrow(
                "select * from view_subject_patch where id = $1 for update",
                patch_id,
            )
            if not p:
                raise NotFoundException()

            changed = {}

            res = await http_client.get(f"https://next.bgm.tv/p1/wiki/subjects/{p['subject_id']}")
            res.raise_for_status()
            original = res.json()

            if p["from_user_id"] != request.auth.user_id:
                raise PermissionDeniedException()

            if p["state"] != PatchState.Pending:
                raise BadRequestException("patch已经被审核")

            for field in ["name", "infobox", "summary"]:
                if getattr(data, field) != original[field]:
                    changed[field] = getattr(data, field)

            nsfw = data.nsfw is not None
            if nsfw != original["nsfw"]:
                changed["nsfw"] = data.nsfw is not None

            if not changed:
                raise BadRequestException("没有实际修改")

            await conn.execute(
                """
            update subject_patch set name=$1, infobox=$2, summary=$3, reason=$4,nsfw=$5,
            original_name=$6, original_infobox=$7,original_summary=$8,updated_at=$9
            where id=$10
            """,
                changed.get("name"),
                changed.get("infobox"),
                changed.get("summary"),
                data.reason,
                changed.get("nsfw"),
                original["name"],
                original["infobox"],
                original["summary"],
                datetime.now(tz=UTC),
                patch_id,
            )

            return Redirect(f"/subject/{patch_id}")


@router
@litestar.get("/suggest-episode")
async def episode_suggest_ui(request: Request, episode_id: int = 0) -> Response[Any]:
    if episode_id == 0:
        return Template("episode/select.html.jinja2")

    if not request.auth:
        request.set_session({session_key_back_to: request.url.path + f"?episode_id={episode_id}"})
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
    description: str

    cf_turnstile_response: str
    reason: str


@router
@litestar.post(
    "/suggest-episode",
    guards=[require_user_login],
    status_code=200,
)
async def creat_episode_patch(
    request: AuthorizedRequest,
    episode_id: int,
    data: Annotated[CreateEpisodePatch, Body(media_type=RequestEncodingType.URL_ENCODED)],
) -> Response[Any]:
    check_invalid_input_str(data.reason)

    if not request.auth.allow_bypass_captcha():
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
        "description": org["summary"],
    }

    keys = ["airdate", "name", "name_cn", "duration", "description"]

    changed = {}

    for key in keys:
        if original_wiki[key] != getattr(data, key):
            changed[key] = getattr(data, key)

    if not changed:
        raise HTTPException("no changes found", status_code=400)

    reason = data.reason.strip()
    if not reason:
        reasons = []
        for key in changed:
            if original_wiki[key]:
                reasons.append(f"修改{patch_keys[key]}")
            else:
                reasons.append(f"添加{patch_keys[key]}")

        reason = "，".join(reasons)

    for key, value in changed.items():
        if c := contains_invalid_input_str(value):
            raise BadRequestException(f"{patch_keys[key]} 包含不可见字符 {c!r}")

    pk = uuid7()

    await pg.execute(
        """
        insert into episode_patch (id, episode_id, from_user_id, reason, original_name, name,
            original_name_cn, name_cn, original_duration, duration,
            original_airdate, airdate, original_description, description, subject_id)
        VALUES ($1, $2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15)
    """,
        pk,
        episode_id,
        request.auth.user_id,
        reason,
        original_wiki["name"],
        changed.get("name"),
        original_wiki["name_cn"],
        changed.get("name_cn"),
        original_wiki["duration"],
        changed.get("duration"),
        original_wiki["airdate"],
        changed.get("airdate"),
        original_wiki["description"],
        changed.get("description"),
        org["subjectID"],
    )

    return Redirect(f"/episode/{pk}")


@router
@litestar.post(
    "/api/delete-episode/{patch_id:uuid}",
    guards=[require_user_login],
    status_code=200,
)
async def delete_episode_patch(patch_id: UUID, request: AuthorizedRequest) -> Redirect:
    async with pg.acquire() as conn:
        async with conn.transaction():
            p = await conn.fetchrow(
                "select from_user_id from view_episode_patch where id = $1",
                patch_id,
            )
            if not p:
                raise NotFoundException()

            if p["from_user_id"] != request.auth.user_id:
                raise NotAuthorizedException("you are not owner of this patch")

            await conn.execute(
                "update episode_patch set deleted_at = $1 where id = $2",
                datetime.now(tz=UTC),
                patch_id,
            )

            return Redirect("/?type=episode")
