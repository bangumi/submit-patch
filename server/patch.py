import difflib
import html
import re
import uuid

import litestar
from litestar.exceptions import InternalServerException, NotFoundException
from litestar.response import Template
from loguru import logger

from server.base import Request, pg
from server.model import Patch, PatchState
from server.router import Router


router = Router()


# from https://stackoverflow.com/a/7160778/8062017
# https// http:// only
is_url_pattern = re.compile(
    r"^https?://"  # http:// or https://
    r"(?:(?:[A-Z0-9](?:[A-Z0-9-]{0,61}[A-Z0-9])?\.)+(?:[A-Z]{2,6}\.?|[A-Z0-9-]{2,}\.?)|"  # domain...
    r"localhost|"  # localhost...d
    r"\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3})"  # ...or ip
    r"(?::\d+)?"  # optional port
    r"(?:/?|[/?]\S+)$",
    re.IGNORECASE,
)


def __render_maybe_url(s: str) -> str:
    if is_url_pattern.match(s):
        escaped = html.escape(s)
        return f'<a href="{escaped}" target="_blank">{escaped}</a>'
    return s


def render_reason(s: str) -> str:
    lines = s.splitlines()

    ss = []

    for line in lines:
        ss.append(" ".join(__render_maybe_url(x) for x in line.split(" ")))

    return "<br>".join(ss)


@router
@litestar.get("/patch/{patch_id:uuid}")
async def get_patch(patch_id: uuid.UUID, request: Request) -> Template:
    p = await pg.fetchrow(
        """select * from patch where id = $1 and deleted_at is NULL limit 1""", patch_id
    )
    if not p:
        raise NotFoundException()

    patch = Patch(**p)

    name_patch = ""
    if patch.name is not None:
        name_patch = "".join(
            difflib.unified_diff([patch.original_name + "\n"], [patch.name + "\n"], "name", "name")
        )

    infobox_patch = ""
    if patch.infobox is not None:
        if patch.original_infobox is None:
            logger.error("broken patch {!r}", patch_id)
            raise InternalServerException
        infobox_patch = "".join(
            difflib.unified_diff(
                (patch.original_infobox + "\n").splitlines(True),
                (patch.infobox + "\n").splitlines(True),
                "infobox",
                "infobox",
                n=5,
            )
        )

    summary_patch = ""
    if patch.summary is not None:
        if patch.original_summary is None:
            logger.error("broken patch {!r}", patch_id)
            raise InternalServerException
        summary_patch = "".join(
            # need a tailing new line to generate correct diff
            difflib.unified_diff(
                (patch.original_summary + "\n").splitlines(True),
                (patch.summary + "\n").splitlines(True),
                "summary",
                "summary",
            )
        )

    reviewer = None
    if patch.state != PatchState.Pending:
        reviewer = await pg.fetchrow(
            "select * from patch_users where user_id=$1", patch.wiki_user_id
        )

    submitter = await pg.fetchrow("select * from patch_users where user_id=$1", patch.from_user_id)

    return Template(
        "patch.html.jinja2",
        context={
            "patch": p,
            "reason": render_reason(p["reason"]),
            "auth": request.auth,
            "name_patch": name_patch,
            "infobox_patch": infobox_patch,
            "summary_patch": summary_patch,
            "reviewer": reviewer,
            "submitter": submitter,
        },
    )


@router
@litestar.get("/episode/{patch_id:uuid}")
async def get_episode_patch(patch_id: uuid.UUID, request: Request) -> Template:
    p = await pg.fetchrow(
        """select * from episode_patch where id = $1 and deleted_at is NULL limit 1""", patch_id
    )
    if not p:
        raise NotFoundException()

    diff = {}

    keys = {
        "name": "标题",
        "name_cn": "简体中文标题",
        "duration": "时长",
        "airdate": "放送日期",
        "description": "简介",
    }

    for key, cn in keys.items():
        after = p[key]
        if after is None:
            continue

        original = p["original_" + key]

        if original != after:
            diff[(key, cn)] = "".join(
                # need a tailing new line to generate correct diff
                difflib.unified_diff(
                    (original + "\n").splitlines(True),
                    (after + "\n").splitlines(True),
                    key,
                    key,
                )
            )

    reviewer = None
    if p["state"] != PatchState.Pending:
        reviewer = await pg.fetchrow(
            "select * from patch_users where user_id=$1", p["wiki_user_id"]
        )

    submitter = await pg.fetchrow("select * from patch_users where user_id=$1", p["from_user_id"])

    return Template(
        "episode/patch.html.jinja2",
        context={
            "patch": p,
            "reason": render_reason(p["reason"]),
            "auth": request.auth,
            "diff": diff,
            "reviewer": reviewer,
            "submitter": submitter,
        },
    )
