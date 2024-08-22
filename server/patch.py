import difflib

import litestar
import uuid6
from litestar.exceptions import InternalServerException, NotFoundException
from litestar.response import Template
from loguru import logger

from server.base import Request, pg
from server.model import Patch, PatchState
from server.router import Router


router = Router()


@router
@litestar.get("/patch/{patch_id:str}")
async def get_patch(patch_id: str, request: Request) -> Template:
    try:
        uuid6.UUID(hex=patch_id)
    except ValueError as e:
        # not valid uuid string, just raise not-found
        raise NotFoundException() from e

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
