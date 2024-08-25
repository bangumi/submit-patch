from typing import Annotated, Any

import litestar
from litestar import params
from litestar.exceptions import NotFoundException
from litestar.response import Redirect, Template

from server.auth import require_user_login
from server.base import BadRequestException, Request, pg
from server.db import fetch_users
from server.model import PatchState, PatchType
from server.router import Router


router = Router()

_page_size = 30


@router
@litestar.get("/", name="index")
async def index(
    request: Request,
    patch_type: Annotated[PatchType, params.Parameter(query="type")] = PatchType.Subject,
    # ?reviewed=0/1/true/false
    # only work on index page
    reviewed: Annotated[bool, params.Parameter(query="reviewed")] = False,
    page: Annotated[int, params.Parameter(query="page", ge=1)] = 1,
) -> litestar.Response[Any]:
    if not request.auth:
        return Template("login.html.jinja2")
    if not request.auth.allow_edit:
        return Redirect(f"/contrib/{request.auth.user_id}")

    if patch_type == PatchType.Subject:
        table = "view_subject_patch"
    elif patch_type == PatchType.Episode:
        table = "view_episode_patch"
    else:
        raise BadRequestException(f"{patch_type} is not valid")

    if not reviewed:
        where = "state = $1"
        order_by = "created_at asc"
    else:
        where = " state != $1"
        order_by = "updated_at desc"

    total: int = await pg.fetchval(
        f"select count(1) from {table} where {where}", PatchState.Pending
    )

    # total=0 -> total_page=1
    # ...
    # total=1 -> total_page=1
    # ...
    # total=100 -> total_page=1

    # total=101 -> total_page=2
    # ...
    # total=200 -> total_page=2

    # total=201 -> total_page=32

    if total == 0:
        total_page = 1
    else:
        total_page = (total + _page_size - 1) // _page_size

    if page > total_page:
        return Redirect(f"/?type={patch_type}&reviewed={int(reviewed)}&page=1")

    if total == 0:
        rows = []
    else:
        rows = await pg.fetch(
            f"select * from {table} where {where} order by {order_by} limit $2 offset $3",
            PatchState.Pending,
            _page_size,
            (page - 1) * _page_size,
        )

    pending_episode = await pg.fetchval(
        "select count(1) from view_episode_patch where state = $1",
        PatchState.Pending,
    )

    pending_subject = await pg.fetchval(
        "select count(1) from view_subject_patch where state = $1",
        PatchState.Pending,
    )

    return Template(
        "list.html.jinja2",
        context={
            "total_page": total_page,
            "current_page": page,
            "rows": rows,
            "filter_reviewed": reviewed,
            "auth": request.auth,
            "users": await fetch_users(rows),
            "patch_type": patch_type,
            "pending_episode": pending_episode,
            "pending_subject": pending_subject,
        },
    )


@router
@litestar.get("/contrib/{user_id:int}", guards=[require_user_login])
async def show_user_contrib(
    user_id: int,
    request: Request,
    patch_type: Annotated[PatchType, params.Parameter(query="type")] = PatchType.Subject,
    page: Annotated[int, params.Parameter(query="page", ge=1)] = 1,
) -> Template:
    if patch_type == PatchType.Subject:
        table = "view_subject_patch"
    elif patch_type == PatchType.Episode:
        table = "view_episode_patch"
    else:
        raise BadRequestException(f"invalid type {patch_type}")

    total = await pg.fetchval(f"select count(1) from {table} where from_user_id = $1", user_id)

    if total == 0:
        total_page = 1
    else:
        total_page = (total + _page_size - 1) // _page_size

    rows = await pg.fetch(
        f"select * from {table} where from_user_id = $1 order by created_at desc limit $2 offset $3",
        user_id,
        _page_size,
        (page - 1) * _page_size,
    )

    nickname = await pg.fetchval("select nickname from patch_users where user_id = $1", user_id)
    if not nickname:
        raise NotFoundException()

    users = await fetch_users(rows)

    return Template(
        "list.html.jinja2",
        context={
            "rows": rows,
            "total_page": total_page,
            "current_page": page,
            "users": users,
            "auth": request.auth,
            "user_id": user_id,
            "patch_type": patch_type,
            "title": f"{nickname} 的历史贡献",
        },
    )


@router
@litestar.get("/review/{user_id:int}", guards=[require_user_login])
async def show_user_review(
    user_id: int,
    request: Request,
    page: Annotated[int, params.Parameter(query="page", ge=1)] = 1,
    patch_type: Annotated[PatchType, params.Parameter(query="type")] = PatchType.Subject,
) -> Template:
    if patch_type == PatchType.Subject:
        table = "view_subject_patch"
    elif patch_type == PatchType.Episode:
        table = "view_episode_patch"
    else:
        raise BadRequestException(f"invalid type {patch_type}")

    total = await pg.fetchval(f"select count(1) from {table} where wiki_user_id = $1", user_id)

    if total == 0:
        total_page = 1
    else:
        total_page = (total + _page_size - 1) // _page_size

    rows = await pg.fetch(
        f"select * from {table} where wiki_user_id = $1 order by created_at desc limit $2 offset $3",
        user_id,
        _page_size,
        (page - 1) * _page_size,
    )

    nickname = await pg.fetchval("select nickname from patch_users where user_id = $1", user_id)
    if not nickname:
        raise NotFoundException()

    users = await fetch_users(rows)

    return Template(
        "list.html.jinja2",
        context={
            "rows": rows,
            "users": users,
            "total_page": total_page,
            "current_page": page,
            "auth": request.auth,
            "user_id": user_id,
            "title": f"{nickname} 的历史审核",
            "patch_type": patch_type,
        },
    )
