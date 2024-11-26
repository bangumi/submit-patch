from dataclasses import dataclass
from datetime import datetime
from typing import Annotated, Any
from uuid import UUID

import litestar
import msgspec.json
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

from common.py.platform import PLATFORM_CONFIG, WIKI_TEMPLATES
from server.auth import require_user_login
from server.base import (
    AuthorizedRequest,
    QueueItem,
    Request,
    external_http_client,
    http_client,
    patch_keys,
    pg,
    session_key_back_to,
    subject_infobox_queue,
)
from server.config import TURNSTILE_SECRET_KEY, UTC
from server.db import create_edit_suggestion
from server.errors import BadRequestException
from server.model import PatchAction, PatchState, PatchType, SubjectPatch, SubjectType
from server.router import Router
from server.strings import check_invalid_input_str, contains_invalid_input_str


router = Router()


async def _validate_captcha(cf_turnstile_response: str) -> None:
    res = await external_http_client.post(
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
    patch_desc: str = ""
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

    if not request.auth.super_user():
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

    pk = await pg.fetchval(
        """
        insert into subject_patch (id, subject_id, from_user_id, reason, name, infobox, summary, nsfw,
                           original_name, original_infobox, original_summary, subject_type, patch_desc)
        VALUES (uuid_generate_v7(), $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
        returning id
    """,
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


@dataclass(slots=True, kw_only=True)
class PartialCreateSubjectPatch:
    name: str | None = None
    infobox: str | None = None
    summary: str | None = None
    nsfw: bool | None = None

    reason: str
    patch_desc: str = ""


@router
@litestar.patch(
    "/suggest-subject",
    guards=[require_user_login],
    status_code=200,
    # opt={"exclude_from_csrf": True},
)
async def suggest_api_from_partial(
    subject_id: int,
    data: PartialCreateSubjectPatch,
    request: AuthorizedRequest,
) -> Redirect:
    data.reason = data.reason.strip()
    data.patch_desc = data.patch_desc.strip()

    if not data.reason:
        raise ValidationException("missing suggestion description")

    check_invalid_input_str(data.reason, data.patch_desc)

    if not request.auth.super_user():
        raise ValidationException(
            "normal users are not allowed to use this api, please contact admin if you need"
        )

    res = await http_client.get(f"https://next.bgm.tv/p1/wiki/subjects/{subject_id}")
    res.raise_for_status()
    original_wiki = res.json()

    original = {}

    changed = {}

    for key in ["name", "infobox", "summary", "nsfw"]:
        before = original_wiki[key]
        after = getattr(data, key)
        if after is None:
            continue
        if before != after:
            changed[key] = after
            original[key] = before

    if not changed:
        raise HTTPException("no changes found", status_code=400)

    for key in ["name", "infobox", "summary"]:
        if key in changed:
            check_invalid_input_str(changed[key])

    pk = await pg.fetchval(
        """
        insert into subject_patch (id, subject_id, from_user_id, reason, name, infobox, summary, nsfw,
                           original_name, original_infobox, original_summary, subject_type, patch_desc)
        VALUES (uuid_generate_v7(), $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
        returning id
    """,
        subject_id,
        request.auth.user_id,
        data.reason,
        changed.get("name"),
        changed.get("infobox"),
        changed.get("summary"),
        changed.get("nsfw"),
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

    patch = SubjectPatch.from_dict(p)

    if patch.from_user_id != request.auth.user_id:
        raise PermissionDeniedException("you are not owner of this patch")

    if patch.state != PatchState.Pending:
        raise BadRequestException("patch 已经被审核")

    if patch.action == PatchAction.Update:
        res = await http_client.get(f"https://next.bgm.tv/p1/wiki/subjects/{p['subject_id']}")
        res.raise_for_status()
        wiki = res.json()
    else:
        wiki = {}

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
    patch_desc: str
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

            patch = SubjectPatch.from_dict(p)

            changed = {}

            if patch.from_user_id != request.auth.user_id:
                raise PermissionDeniedException()

            if patch.state != PatchState.Pending:
                raise BadRequestException("patch已经被审核")

            if patch.action == PatchAction.Create:
                await conn.execute(
                    """
                update subject_patch set name=$1, infobox=$2, summary=$3, nsfw=$4, reason=$5,
                updated_at=$6
                where id=$7
                """,
                    data.name,
                    data.infobox,
                    data.summary,
                    data.nsfw is not None,
                    data.reason,
                    datetime.now(tz=UTC),
                    patch_id,
                )

                return Redirect(f"/subject/{patch_id}")

            res = await http_client.get(f"https://next.bgm.tv/p1/wiki/subjects/{p['subject_id']}")
            res.raise_for_status()
            original = res.json()

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
            original_name=$6, original_infobox=$7,original_summary=$8,updated_at=$9,patch_desc=$10
            where id=$11
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
                data.patch_desc,
                patch_id,
            )

            await create_edit_suggestion(
                conn, patch_id, PatchType.Subject, text="提交者进行了修改", from_user=0
            )

            if "infobox" in changed:
                await subject_infobox_queue.put(
                    QueueItem(
                        patch_id=patch_id,
                        infobox=changed["infobox"],
                    )
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

    if not request.auth.super_user():
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

    pk = await pg.fetchval(
        """
        insert into episode_patch (id, episode_id, from_user_id, reason, original_name, name,
            original_name_cn, name_cn, original_duration, duration,
            original_airdate, airdate, original_description, description, subject_id, ep)
        VALUES (uuid_generate_v7(), $1, $2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15)
        returning id
    """,
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
        org["ep"],
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


@router
@litestar.get(
    "/new-subject/{subject_type:int}",
    guards=[require_user_login],
)
async def new_subject(subject_type: SubjectType) -> Template:
    return Template(
        "new-subject.html.jinja2",
        context={
            "subject_type": subject_type,
            "pp": PLATFORM_CONFIG.get(subject_type),
            "platforms": msgspec.json.encode(PLATFORM_CONFIG.get(subject_type)).decode(),
            "templates": WIKI_TEMPLATES,
        },
    )


@dataclass(frozen=True, slots=True, kw_only=True)
class NewSubject:
    name: str
    type_id: SubjectType
    platform: int
    infobox: str
    summary: str
    # HTML form will only include checkbox when it's checked,
    # so any input is true, default value is false.
    nsfw: str | None = None

    patch_desc: str = ""
    cf_turnstile_response: str


@router
@litestar.post(
    "/new-subject",
    guards=[require_user_login],
)
async def patch_for_new_subject(
    data: Annotated[NewSubject, Body(media_type=RequestEncodingType.URL_ENCODED)],
    request: AuthorizedRequest,
) -> Redirect:
    if not request.auth.super_user():
        await _validate_captcha(data.cf_turnstile_response)

    if data.platform not in PLATFORM_CONFIG.get(data.type_id, {}):
        raise BadRequestException("平台不正确")

    pk = await pg.fetchval(
        """
        insert into subject_patch (id, subject_id, from_user_id, reason, name, infobox, summary, nsfw,
                             subject_type, platform, patch_desc, action, original_name)
        VALUES (uuid_generate_v7(), 0, $1, '', $2, $3, $4, $5, $6, $7, $8, $9, '')
        returning id
    """,
        request.auth.user_id,
        data.name,
        data.infobox,
        data.summary,
        data.nsfw is not None,
        data.type_id,
        data.platform,
        data.patch_desc,
        PatchAction.Create,
    )

    return Redirect(f"/subject/{pk}")
