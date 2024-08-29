import difflib
import uuid

import litestar
from litestar.exceptions import InternalServerException, NotFoundException
from litestar.response import Redirect, Template
from loguru import logger

from server.base import Request, patch_keys, pg
from server.model import PatchState, PatchType, SubjectPatch
from server.router import Router
from server.strings import invisible_escape


router = Router()


@router
@litestar.get("/patch/{patch_id:uuid}", sync_to_thread=False)
def get_patch_redirect(patch_id: uuid.UUID) -> Redirect:
    return Redirect(f"/subject/{patch_id}")


@router
@litestar.get("/subject/{patch_id:uuid}")
async def get_patch(patch_id: uuid.UUID, request: Request) -> Template:
    p = await pg.fetchrow(
        """select * from subject_patch where id = $1 and deleted_at is NULL limit 1""", patch_id
    )
    if not p:
        raise NotFoundException()

    patch = SubjectPatch(**p)

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
                (invisible_escape(patch.original_infobox) + "\n").splitlines(True),
                (invisible_escape(patch.infobox) + "\n").splitlines(True),
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

    suggestions = await pg.fetch(
        """
    select * from edit_suggestion
        inner join patch_users on patch_users.user_id = edit_suggestion.from_user
        where deleted_at IS NULL AND
            patch_id = $1 AND
            patch_type = $2
        order by created_at
    """,
        patch_id,
        PatchType.Subject,
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
            "reason": p["reason"],
            "auth": request.auth,
            "suggestions": suggestions,
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

    for key in patch_keys:
        after = p[key]
        if after is None:
            continue

        original = p["original_" + key]

        if original != after:
            if key != "description":
                diff[key] = "".join(
                    # need a tailing new line to generate correct diff
                    difflib.unified_diff(
                        [original + "\n"],
                        [after + "\n"],
                        key,
                        key,
                    )
                )
            else:
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
            "reason": p["reason"],
            "auth": request.auth,
            "keys": patch_keys,
            "diff": diff,
            "reviewer": reviewer,
            "submitter": submitter,
        },
    )
