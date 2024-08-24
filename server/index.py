import enum
from typing import Annotated, Any

import asyncpg
import litestar
from litestar.exceptions import NotFoundException
from litestar.params import Parameter
from litestar.response import Redirect, Template
from sqlalchemy import func, literal_column, null, select

from server import db
from server.auth import require_user_login
from server.base import BadRequestException, Request, async_session, pg
from server.model import PatchState
from server.router import Router


router = Router()


class PatchType(str, enum.Enum):
    Subject = "subject"
    Episode = "episode"

    def __str__(self) -> str:
        return self.value


_page_size = 100


@router
@litestar.get("/", name="index")
async def index(
    request: Request,
    patch_type: Annotated[PatchType, Parameter(query="type")] = PatchType.Subject,
    # ?reviewed=0/1/true/false
    # only work on index page
    reviewed: Annotated[bool, Parameter(query="reviewed")] = False,
    page: Annotated[int, Parameter(query="page", ge=1)] = 1,
) -> litestar.Response[Any]:
    if not request.auth:
        return Template("login.html.jinja2")
    if not request.auth.allow_edit:
        return Redirect(f"/contrib/{request.auth.user_id}")

    if patch_type == PatchType.Subject:
        sa_table: type[db.BasePatchMixin] = db.SubjectPatch
    elif patch_type == PatchType.Episode:
        sa_table = db.EpisodePatch
    else:
        raise BadRequestException(f"{patch_type} is not valid")

    where = [sa_table.deleted_at == null()]
    if reviewed:
        where.append(sa_table.state != PatchState.Pending)
        order_by = sa_table.created_at
    else:
        where.append(sa_table.state == PatchState.Pending)
        order_by = sa_table.created_at.desc()

    async with async_session() as session:
        total = await session.scalar(
            select(func.count(literal_column("1"))).select_from(sa_table).where(*where)
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
            rows = []
        else:
            result = await session.execute(
                select(sa_table)
                .where(*where)
                .limit(_page_size)
                .offset((page - 1) * _page_size)
                .order_by(order_by)
            )

            rows = list(result.scalars())

    pending_episode = await pg.fetchval(
        "select count(1) from episode_patch where deleted_at is NULL and state = $1",
        PatchState.Pending,
    )

    pending_subject = await pg.fetchval(
        "select count(1) from patch where deleted_at is NULL and state = $1",
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
            "users": await __fetch_users_of_db(rows),
            "patch_type": patch_type,
            "pending_episode": pending_episode,
            "pending_subject": pending_subject,
        },
    )


async def __fetch_users_of_db(rows: list[db.BasePatchMixin]) -> dict[int, asyncpg.Record]:
    user_id = {x.from_user_id for x in rows} | {x.wiki_user_id for x in rows}
    user_id.discard(None)
    user_id.discard(0)

    users = {
        x["user_id"]: x
        for x in await pg.fetch("select * from patch_users where user_id = any($1)", user_id)
    }

    return users


async def __fetch_users(rows: list[asyncpg.Record]) -> dict[int, asyncpg.Record]:
    user_id = {x["from_user_id"] for x in rows} | {x["wiki_user_id"] for x in rows}
    user_id.discard(None)
    user_id.discard(0)

    users = {
        x["user_id"]: x
        for x in await pg.fetch("select * from patch_users where user_id = any($1)", user_id)
    }

    return users


@router
@litestar.get("/contrib/{user_id:int}", guards=[require_user_login])
async def show_user_contrib(
    user_id: int,
    request: Request,
    patch_type: Annotated[PatchType, Parameter(query="type")] = PatchType.Subject,
) -> Template:
    if patch_type == PatchType.Subject:
        rows = await pg.fetch(
            "select * from patch where from_user_id = $1 and deleted_at is NULL order by created_at desc",
            user_id,
        )
    elif patch_type == PatchType.Episode:
        rows = await pg.fetch(
            "select * from episode_patch where from_user_id = $1 and deleted_at is NULL order by created_at desc",
            user_id,
        )
    else:
        raise BadRequestException(f"invalid type {patch_type}")

    nickname = await pg.fetchval("select nickname from patch_users where user_id = $1", user_id)
    if not nickname:
        raise NotFoundException()

    users = await __fetch_users(rows)

    return Template(
        "list.html.jinja2",
        context={
            "rows": rows,
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
    patch_type: Annotated[PatchType, Parameter(query="type")] = PatchType.Subject,
) -> Template:
    if patch_type == PatchType.Subject:
        rows = await pg.fetch(
            "select * from patch where wiki_user_id = $1 and deleted_at is NULL order by created_at desc",
            user_id,
        )
    elif patch_type == PatchType.Episode:
        rows = await pg.fetch(
            "select * from episode_patch where wiki_user_id = $1 and deleted_at is NULL order by created_at desc",
            user_id,
        )
    else:
        raise BadRequestException(f"invalid type {patch_type}")

    nickname = await pg.fetchval("select nickname from patch_users where user_id = $1", user_id)
    if not nickname:
        raise NotFoundException()

    users = await __fetch_users(rows)

    return Template(
        "list.html.jinja2",
        context={
            "rows": rows,
            "users": users,
            "auth": request.auth,
            "user_id": user_id,
            "title": f"{nickname} 的历史审核",
            "patch_type": patch_type,
        },
    )
