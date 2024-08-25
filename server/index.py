import enum
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

html_patch_state_filter = [
    ("pending", "待审核"),
    ("reviewed", "已审核"),
    ("all", "全部"),
    ("rejected", "拒绝"),
    ("accepted", "接受"),
]


@enum.unique
class ReviewedStateFilter(str, enum.Enum):
    All = "all"
    Rejected = "rejected"
    Accepted = "accepted"

    def to_sql(self, index: int = 1) -> tuple[str, Any]:
        match self.value:
            case "all":
                return f"1=${index}", 1
            case "rejected":
                return f"state = ${index}", PatchState.Rejected
            case "accepted":
                return f"state = ${index}", PatchState.Accept

        raise NotImplementedError()


@enum.unique
class StateFilter(str, enum.Enum):
    All = "all"
    Pending = "pending"
    Reviewed = "reviewed"
    Rejected = "rejected"
    Accepted = "accepted"

    def __str__(self) -> str:
        return self.value

    def to_sql(self, index: int = 1) -> tuple[str, Any]:
        match self.value:
            case "all":
                return f"1=${index}", 1
            case "pending":
                return f"state = ${index}", PatchState.Pending
            case "reviewed":
                return f"state != ${index}", PatchState.Pending
            case "rejected":
                return f"state = ${index}", PatchState.Rejected
            case "accepted":
                return f"state = ${index}", PatchState.Accept

        raise NotImplementedError()


@router
@litestar.get("/", name="index")
async def _(
    request: Request,
    patch_type: Annotated[PatchType, params.Parameter(query="type")] = PatchType.Subject,
    # ?reviewed=0/1/true/false
    # only work on index page
    patch_state_filter: Annotated[
        StateFilter, params.Parameter(query="state")
    ] = StateFilter.Pending,
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

    where, arg = patch_state_filter.to_sql(index=1)
    if patch_state_filter == StateFilter.Pending:
        order_by = "created_at asc"
    elif patch_state_filter == StateFilter.All:
        order_by = "created_at desc"
    else:
        order_by = "updated_at desc"

    total: int = await pg.fetchval(f"select count(1) from {table} where {where}", arg)

    total_page = (total + _page_size - 1) // _page_size

    if page > total_page:
        return Redirect(f"/?type={patch_type}&state={patch_state_filter}&page=1")

    if total == 0:
        rows = []
    else:
        rows = await pg.fetch(
            f"select * from {table} where {where} order by {order_by} limit $2 offset $3",
            arg,
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
            "patch_state_filter": html_patch_state_filter,
            "current_state": patch_state_filter,
            "current_page": page,
            "rows": rows,
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
    request: Request,
    user_id: int,
    *,
    patch_state_filter: Annotated[StateFilter, params.Parameter(query="state")] = StateFilter.All,
    patch_type: Annotated[PatchType, params.Parameter(query="type")] = PatchType.Subject,
    page: Annotated[int, params.Parameter(query="page", ge=1)] = 1,
) -> Template:
    if patch_type == PatchType.Subject:
        table = "view_subject_patch"
    elif patch_type == PatchType.Episode:
        table = "view_episode_patch"
    else:
        raise NotImplementedError()

    where, arg = patch_state_filter.to_sql(index=2)

    total = await pg.fetchval(
        f"select count(1) from {table} where from_user_id = $1 AND {where}",
        user_id,
        arg,
    )

    total_page = (total + _page_size - 1) // _page_size

    rows = await pg.fetch(
        f"select * from {table} where from_user_id = $1 AND {where} order by created_at desc limit $3 offset $4",
        user_id,
        arg,
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
            "current_state": patch_state_filter,
            "patch_state_filter": html_patch_state_filter,
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
    request: Request,
    user_id: int,
    *,
    patch_state_filter: Annotated[
        ReviewedStateFilter, params.Parameter(query="state")
    ] = ReviewedStateFilter.All,
    page: Annotated[int, params.Parameter(query="page", ge=1)] = 1,
    patch_type: Annotated[PatchType, params.Parameter(query="type")] = PatchType.Subject,
) -> Template:
    if patch_type == PatchType.Subject:
        table = "view_subject_patch"
    elif patch_type == PatchType.Episode:
        table = "view_episode_patch"
    else:
        raise BadRequestException(f"invalid type {patch_type}")

    where, arg = patch_state_filter.to_sql(index=2)

    total = await pg.fetchval(
        f"select count(1) from {table} where wiki_user_id = $1 AND {where}",
        user_id,
        arg,
    )

    total_page = (total + _page_size - 1) // _page_size

    rows = await pg.fetch(
        f"select * from {table} where wiki_user_id = $1 AND {where} order by created_at desc limit $3 offset $4",
        user_id,
        arg,
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
            "patch_state_filter": html_patch_state_filter[2:],
            "skip_pending": True,
            "total_page": total_page,
            "current_page": page,
            "current_state": patch_state_filter,
            "auth": request.auth,
            "user_id": user_id,
            "title": f"{nickname} 的历史审核",
            "patch_type": patch_type,
        },
    )
