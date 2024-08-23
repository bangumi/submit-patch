import difflib
import uuid

import litestar
from litestar.exceptions import InternalServerException, NotFoundException
from litestar.response import Template
from loguru import logger

from server.base import Request, pg
from server.model import Patch, PatchState
from server.router import Router


router = Router()


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

    keys = ["name", "name_cn", "duration", "airdate", "description"]

    for key in keys:
        after = p[key]
        if after is None:
            continue

        original = p["original_" + key]

        if original != after:
            diff[key] = "".join(
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
            "auth": request.auth,
            "diff": diff,
            "diff_key_cn": {
                "name": "标题",
                "name_cn": "简体中文标题",
                "duration": "时长",
                "airdate": "放送日期",
                "description": "简介",
            },
            "reviewer": reviewer,
            "submitter": submitter,
        },
    )
