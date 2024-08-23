import asyncio
import enum
import html
import mimetypes
import os
from datetime import datetime
from pathlib import Path
from typing import Annotated, Any, NamedTuple

import asyncpg
import litestar
from litestar import Response
from litestar.config.csrf import CSRFConfig
from litestar.contrib.jinja import JinjaTemplateEngine
from litestar.datastructures import State
from litestar.exceptions import (
    HTTPException,
    NotFoundException,
)
from litestar.params import Parameter
from litestar.response import Template
from litestar.static_files import create_static_files_router
from litestar.status_codes import HTTP_500_INTERNAL_SERVER_ERROR
from litestar.stores.redis import RedisStore
from litestar.template import TemplateConfig
from loguru import logger

from config import (
    CSRF_SECRET_TOKEN,
    DEV,
    PROJECT_PATH,
    UTC,
)
from server import auth, contrib, patch, review, tmpl
from server.auth import require_user_login, session_auth_config
from server.base import BadRequestException, Request, http_client, pg, pg_pool_startup, redis_client
from server.migration import run_migration
from server.model import PatchState
from server.router import Router


router = Router()


class File(NamedTuple):
    content: bytes
    content_type: str | None


static_path = PROJECT_PATH.joinpath("server/static/")
if not DEV:
    static_files: dict[str, File] = {}

    for top, _, files in os.walk(static_path):
        for file in files:
            file_path = Path(top, file)
            rel_path = file_path.relative_to(static_path).as_posix()
            static_files["/" + rel_path] = File(
                content=file_path.read_bytes(), content_type=mimetypes.guess_type(file)[0]
            )

    @router
    @litestar.get("/static/{fp:path}", sync_to_thread=False)
    def static_file_handler(fp: str) -> Response[bytes]:
        try:
            f = static_files[fp]
            return Response(
                content=f.content,
                media_type=f.content_type,
                headers={"cache-control": "max-age=1200"},
            )
        except KeyError:
            raise NotFoundException()  # noqa: B904

else:

    router(
        create_static_files_router(
            path="/static/",
            directories=[static_path],
            send_as_attachment=False,
            html_mode=False,
        )
    )


async def __fetch_users(rows: list[asyncpg.Record]) -> dict[int, asyncpg.Record]:
    user_id = {x["from_user_id"] for x in rows} | {x["wiki_user_id"] for x in rows}
    user_id.discard(None)
    user_id.discard(0)

    users = {
        x["user_id"]: x
        for x in await pg.fetch("select * from patch_users where user_id = any($1)", user_id)
    }

    return users


class PatchType(str, enum.Enum):
    Subject = "subject"
    Episode = "episode"


@router
@litestar.get("/")
async def index(
    request: Request, patch_type: Annotated[PatchType, Parameter(query="type")] = PatchType.Subject
) -> Template:
    if not request.auth:
        return Template("login.html.jinja2")

    if patch_type == PatchType.Subject:
        if not request.auth.allow_edit:
            rows = await pg.fetch(
                "select * from patch where from_user_id = $1 and deleted_at is NULL order by created_at desc",
                request.auth.user_id,
            )
        else:
            rows = await pg.fetch(
                """
                select * from patch where deleted_at is NULL and state = $1
                union
                (select * from patch  where deleted_at is NULL and state != $1 order by updated_at desc limit 10)
                """,
                PatchState.Pending,
            )

            rows.sort(key=__index_row_sorter, reverse=True)

    elif patch_type == PatchType.Episode:
        if not request.auth.allow_edit:
            rows = await pg.fetch(
                "select * from episode_patch where from_user_id = $1 and deleted_at is NULL order by created_at desc",
                request.auth.user_id,
            )
        else:
            rows = await pg.fetch(
                """
                select * from episode_patch where deleted_at is NULL and state = $1
                union
                (select * from episode_patch  where deleted_at is NULL and state != $1 order by updated_at desc limit 10)
                """,
                PatchState.Pending,
            )

            rows.sort(key=__index_row_sorter, reverse=True)
    else:
        raise BadRequestException(f"{patch_type} is not valid")

    return Template(
        "list.html.jinja2",
        context={
            "rows": rows,
            "auth": request.auth,
            "users": await __fetch_users(rows),
            "patch_type": patch_type,
        },
    )


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

    users = await __fetch_users(rows)

    return Template(
        "list.html.jinja2",
        context={
            "rows": rows,
            "users": users,
            "auth": request.auth,
            "user_id": user_id,
            "patch_type": patch_type,
            "title": f"{users[user_id]['nickname']} 的历史贡献",
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
            "select * from episode_patch where from_user_id = $1 and deleted_at is NULL order by created_at desc",
            user_id,
        )
    else:
        raise BadRequestException(f"invalid type {patch_type}")

    users = await __fetch_users(rows)

    return Template(
        "list.html.jinja2",
        context={
            "rows": rows,
            "users": users,
            "auth": request.auth,
            "user_id": user_id,
            "title": f"{users[user_id]['nickname']} 的历史审核",
            "patch_type": patch_type,
        },
    )


def __index_row_sorter(r: asyncpg.Record) -> tuple[int, datetime]:
    if r["state"] == PatchState.Pending:
        return 1, r["created_at"]

    return 0, r["updated_at"]


def before_req(req: litestar.Request[None, None, State]) -> None:
    req.state["now"] = datetime.now(tz=UTC)


def plain_text_exception_handler(_: Request, exc: HTTPException) -> Template:
    """Default handler for exceptions subclassed from HTTPException."""
    return Template(
        "error.html.jinja2",
        status_code=exc.status_code,
        context={"error": exc, "detail": exc.detail},
    )


def internal_error_handler(_: Request, exc: Exception) -> Response[Any]:
    logger.error("internal server error: {}", exc)
    return Response(
        content={"status_code": 500, "detail": "Internal Server Error"},
        status_code=HTTP_500_INTERNAL_SERVER_ERROR,
    )


async def startup_fetch_missing_users() -> None:
    logger.info("fetch missing users")
    s = set()

    for u1, u2 in await pg.fetch("select from_user_id, wiki_user_id from patch"):
        s.add(u1)
        s.add(u2)

    for u1, u2 in await pg.fetch("select from_user_id, wiki_user_id from episode_patch"):
        s.add(u1)
        s.add(u2)

    s.discard(None)
    s.discard(0)

    user_fetched = [
        x[0] for x in await pg.fetch("select user_id from patch_users where user_id = any($1)", s)
    ]

    s = {x for x in s if x not in user_fetched}
    if not s:
        return

    for user in s:
        r = await http_client.get(f"https://api.bgm.tv/user/{user}")
        data = r.json()
        await pg.execute(
            """
            insert into patch_users (user_id, username, nickname) VALUES ($1, $2, $3)
            on conflict (user_id) do update set
                username = excluded.username,
                nickname = excluded.nickname
        """,
            data["id"],
            data["username"],
            html.unescape(data["nickname"]),
        )


@router
@litestar.get("/badge.svg")
async def badge() -> Response[bytes]:
    key = "patch:rest:pending"
    pending = await redis_client.get(key)

    if pending is not None:
        return Response(pending, media_type="image/svg+xml")

    rest = sum(
        await asyncio.gather(
            pg.fetchval(
                "select count(1) from patch where deleted_at IS NULL and state = $1",
                PatchState.Pending,
            ),
            pg.fetchval(
                "select count(1) from episode_patch where deleted_at IS NULL and state = $1",
                PatchState.Pending,
            ),
        )
    )

    if rest == 0:
        # https://img.shields.io/badge/%E5%BE%85%E5%AE%A1%E6%A0%B8-0-green
        res = """
<svg xmlns="http://www.w3.org/2000/svg" xmlns:xlink="http://www.w3.org/1999/xlink"
     width="60" height="20" role="img" aria-label="待审核: 0">
    <title>待审核: 0</title>
    <linearGradient id="s" x2="0" y2="100%">
        <stop offset="0" stop-color="#bbb" stop-opacity=".1"/>
        <stop offset="1" stop-opacity=".1"/>
    </linearGradient>
    <clipPath id="r">
        <rect width="60" height="20" rx="3" fill="#fff"/>
    </clipPath>
    <g clip-path="url(#r)">
        <rect width="43" height="20" fill="#555"/>
        <rect x="43" width="17" height="20" fill="#97ca00"/>
        <rect width="60" height="20" fill="url(#s)"/>
    </g>
    <g fill="#fff" text-anchor="middle"
       font-family="Verdana,Geneva,DejaVu Sans,sans-serif"
       text-rendering="geometricPrecision" font-size="110">
        <text aria-hidden="true" x="225" y="150" fill="#010101" fill-opacity=".3"
              transform="scale(.1)" textLength="330">待审核
        </text>
        <text x="225" y="140" transform="scale(.1)" fill="#fff" textLength="330">
            待审核
        </text>
        <text aria-hidden="true" x="505" y="150" fill="#010101" fill-opacity=".3"
              transform="scale(.1)" textLength="70">0
        </text>
        <text x="505" y="140" transform="scale(.1)" fill="#fff" textLength="70">0</text>
    </g>
</svg>
        """.encode()
    elif rest < 10:
        # https://img.shields.io/badge/%E5%BE%85%E5%AE%A1%E6%A0%B8-8-orange
        res = f"""
<svg xmlns="http://www.w3.org/2000/svg" xmlns:xlink="http://www.w3.org/1999/xlink"
     width="60" height="20" role="img" aria-label="待审核: 8">
    <title>待审核: 8</title>
    <linearGradient id="s" x2="0" y2="100%">
        <stop offset="0" stop-color="#bbb" stop-opacity=".1"/>
        <stop offset="1" stop-opacity=".1"/>
    </linearGradient>
    <clipPath id="r">
        <rect width="60" height="20" rx="3" fill="#fff"/>
    </clipPath>
    <g clip-path="url(#r)">
        <rect width="43" height="20" fill="#555"/>
        <rect x="43" width="17" height="20" fill="#007ec6"/>
        <rect width="60" height="20" fill="url(#s)"/>
    </g>
    <g fill="#fff" text-anchor="middle"
       font-family="Verdana,Geneva,DejaVu Sans,sans-serif"
       text-rendering="geometricPrecision" font-size="110">
        <text aria-hidden="true" x="225" y="150" fill="#010101" fill-opacity=".3"
              transform="scale(.1)" textLength="330">待审核
        </text>
        <text x="225" y="140" transform="scale(.1)" fill="#fff" textLength="330">
            待审核
        </text>
        <text aria-hidden="true" x="505" y="150" fill="#010101" fill-opacity=".3"
              transform="scale(.1)" textLength="70">8
        </text>
        <text x="505" y="140" transform="scale(.1)" fill="#fff" textLength="70">
            {rest}
        </text>
    </g>
</svg>
        """.encode()
    elif rest < 100:
        # https://img.shields.io/badge/%E5%BE%85%E5%AE%A1%E6%A0%B8-50-oragan
        res = f"""
<svg xmlns="http://www.w3.org/2000/svg" xmlns:xlink="http://www.w3.org/1999/xlink"
     width="66" height="20" role="img" aria-label="待审核: 50">
    <title>待审核: 50</title>
    <linearGradient id="s" x2="0" y2="100%">
        <stop offset="0" stop-color="#bbb" stop-opacity=".1"/>
        <stop offset="1" stop-opacity=".1"/>
    </linearGradient>
    <clipPath id="r">
        <rect width="66" height="20" rx="3" fill="#fff"/>
    </clipPath>
    <g clip-path="url(#r)">
        <rect width="43" height="20" fill="#555"/>
        <rect x="43" width="23" height="20" fill="#fe7d37"/>
        <rect width="66" height="20" fill="url(#s)"/>
    </g>
    <g fill="#fff" text-anchor="middle"
       font-family="Verdana,Geneva,DejaVu Sans,sans-serif"
       text-rendering="geometricPrecision" font-size="110">
        <text aria-hidden="true" x="225" y="150" fill="#010101" fill-opacity=".3"
              transform="scale(.1)" textLength="330">待审核
        </text>
        <text x="225" y="140" transform="scale(.1)" fill="#fff" textLength="330">
            待审核
        </text>
        <text aria-hidden="true" x="535" y="150" fill="#010101" fill-opacity=".3"
              transform="scale(.1)" textLength="130">50
        </text>
        <text x="535" y="140" transform="scale(.1)" fill="#fff" textLength="130">
            {rest}
        </text>
    </g>
</svg>
        """.encode()
    else:
        # https://img.shields.io/badge/待审核->100-red
        res = """
<svg xmlns="http://www.w3.org/2000/svg" xmlns:xlink="http://www.w3.org/1999/xlink"
     width="82" height="20" role="img" aria-label="待审核: &gt;100">
    <title>待审核: &gt;100</title>
    <linearGradient id="s" x2="0" y2="100%">
        <stop offset="0" stop-color="#bbb" stop-opacity=".1"/>
        <stop offset="1" stop-opacity=".1"/>
    </linearGradient>
    <clipPath id="r">
        <rect width="82" height="20" rx="3" fill="#fff"/>
    </clipPath>
    <g clip-path="url(#r)">
        <rect width="43" height="20" fill="#555"/>
        <rect x="43" width="39" height="20" fill="#e05d44"/>
        <rect width="82" height="20" fill="url(#s)"/>
    </g>
    <g fill="#fff" text-anchor="middle"
       font-family="Verdana,Geneva,DejaVu Sans,sans-serif"
       text-rendering="geometricPrecision" font-size="110">
        <text aria-hidden="true" x="225" y="150" fill="#010101" fill-opacity=".3"
              transform="scale(.1)" textLength="330">待审核
        </text>
        <text x="225" y="140" transform="scale(.1)" fill="#fff" textLength="330">
            待审核
        </text>
        <text aria-hidden="true" x="615" y="150" fill="#010101" fill-opacity=".3"
              transform="scale(.1)" textLength="290">&gt;100
        </text>
        <text x="615" y="140" transform="scale(.1)" fill="#fff" textLength="290">
            &gt;100
        </text>
    </g>
</svg>
        """.encode()

    await redis_client.set(key, res, ex=10)

    return Response(res, media_type="image/svg+xml")


app = litestar.Litestar(
    [
        *auth.router,
        *contrib.router,
        *review.router,
        *patch.router,
        *router,
    ],
    template_config=TemplateConfig(
        engine=JinjaTemplateEngine.from_environment(tmpl.engine),
    ),
    stores={"sessions": RedisStore(redis_client, handle_client_shutdown=False)},
    on_startup=[pg_pool_startup, run_migration, startup_fetch_missing_users],
    csrf_config=CSRFConfig(secret=CSRF_SECRET_TOKEN, cookie_name="s-csrf-token"),
    before_request=before_req,
    middleware=[session_auth_config.middleware],
    exception_handlers={
        HTTPException: plain_text_exception_handler,
        Exception: internal_error_handler,
    },
    debug=DEV,
)
