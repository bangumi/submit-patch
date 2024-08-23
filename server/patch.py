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

    name_patch = __try_diff(patch_id, patch.original_name, patch.name, "name")
    infobox_patch = __try_diff(patch_id, patch.original_infobox, patch.infobox, "infobox", n=5)
    summary_patch = __try_diff(patch_id, patch.original_summary, patch.summary, "summary")

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


def __try_diff(
    patch_id: uuid.UUID, before: str | None, after: str | None, name: str, **kwargs
) -> str:
    if after is None:
        return ""
    if before is None:
        logger.error("broken patch {!r}", patch_id)
        raise InternalServerException
    return "".join(
        # need a tailing new line to generate correct diff
        difflib.unified_diff(
            (before + "\n").splitlines(True),
            (after + "\n").splitlines(True),
            name,
            name,
            **kwargs,
        )
    )


@router
@litestar.get("/episode/{patch_id:uuid}")
async def get_episode_patch(patch_id: uuid.UUID, request: Request) -> Template:
    p = await pg.fetchrow(
        """select * from episode_patch where id = $1 and deleted_at is NULL limit 1""", patch_id
    )
    if not p:
        raise NotFoundException()

    diff = []

    keys = ["name", "name_cn", "duration", "airdate", "description"]

    for key in keys:
        after = p[key]
        if after is None:
            continue

        original = p["original_" + key]

        if original != after:
            diff.append(
                "".join(
                    # need a tailing new line to generate correct diff
                    difflib.unified_diff(
                        (original + "\n").splitlines(True),
                        (after + "\n").splitlines(True),
                        key,
                        key,
                    )
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
            "diff": "".join(diff),
            "reviewer": reviewer,
            "submitter": submitter,
        },
    )
