import difflib
import html
import mimetypes
import os
from datetime import datetime
from pathlib import Path
from typing import Any, NamedTuple

import asyncpg
import litestar
import uuid6
from litestar import Response
from litestar.config.csrf import CSRFConfig
from litestar.contrib.jinja import JinjaTemplateEngine
from litestar.datastructures import State
from litestar.exceptions import (
    HTTPException,
    InternalServerException,
    NotFoundException,
)
from litestar.response import Template
from litestar.status_codes import HTTP_500_INTERNAL_SERVER_ERROR
from litestar.stores.redis import RedisStore
from litestar.template import TemplateConfig
from loguru import logger
from redis.asyncio.client import Redis

from config import (
    CSRF_SECRET_TOKEN,
    DEV,
    PROJECT_PATH,
    REDIS_DSN,
    UTC,
)
from server import tmpl
from server.auth import callback, login, require_user_login, session_auth_config
from server.base import Request, http_client, pg, pg_pool_startup
from server.contrib import delete_patch, suggest_api, suggest_ui
from server.model import Patch, PatchState
from server.review import review_patch


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

    @litestar.get("/static/{fp:path}", sync_to_thread=True)
    def static_file_handler(fp: str) -> Response[bytes]:
        # fp is '/...', so we need to remove prefix make it relative
        return Response(
            static_path.joinpath(fp[1:]).read_bytes(), media_type=mimetypes.guess_type(fp)[0]
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


@litestar.get("/")
async def index(request: Request) -> Template:
    if not request.auth:
        return Template("login.html.jinja2")

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

    return Template(
        "list.html.jinja2",
        context={"rows": rows, "auth": request.auth, "users": await __fetch_users(rows)},
    )


@litestar.get("/contrib/{user_id:int}", guards=[require_user_login])
async def show_user_contrib(user_id: int, request: Request) -> Template:
    rows = await pg.fetch(
        "select * from patch where from_user_id = $1 and deleted_at is NULL order by created_at desc",
        user_id,
    )

    users = await __fetch_users(rows)

    return Template(
        "list.html.jinja2",
        context={
            "rows": rows,
            "users": users,
            "auth": request.auth,
            "user_id": user_id,
            "title": f"{users[user_id]['nickname']} 的历史贡献",
        },
    )


@litestar.get("/review/{user_id:int}", guards=[require_user_login])
async def show_user_review(user_id: int, request: Request) -> Template:
    rows = await pg.fetch(
        "select * from patch where wiki_user_id = $1 and deleted_at is NULL order by created_at desc",
        user_id,
    )

    users = await __fetch_users(rows)

    return Template(
        "list.html.jinja2",
        context={
            "rows": rows,
            "users": users,
            "auth": request.auth,
            "user_id": user_id,
            "title": f"{users[user_id]['nickname']} 的历史审核",
        },
    )


def __index_row_sorter(r: asyncpg.Record) -> tuple[int, datetime]:
    if r["state"] == PatchState.Pending:
        return 1, r["created_at"]

    return 0, r["updated_at"]


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
                patch.original_infobox.splitlines(True),
                patch.infobox.splitlines(True),
                "infobox",
                "infobox",
            )
        )

    summary_patch = ""
    if patch.summary is not None:
        if patch.original_summary is None:
            logger.error("broken patch {!r}", patch_id)
            raise InternalServerException
        summary_patch = "".join(
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


async def startup_fetch_missing_users(*args: Any, **kwargs: Any) -> None:
    logger.info("fetch missing users")
    results = await pg.fetch("select from_user_id, wiki_user_id from patch")
    s = set()
    for u1, u2 in results:
        s.add(u1)
        s.add(u2)

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


app = litestar.Litestar(
    [
        index,
        show_user_review,
        show_user_contrib,
        login,
        callback,
        suggest_ui,
        suggest_api,
        get_patch,
        delete_patch,
        review_patch,
        static_file_handler,
    ],
    template_config=TemplateConfig(
        engine=JinjaTemplateEngine.from_environment(tmpl.engine),
    ),
    stores={"sessions": RedisStore(Redis.from_url(REDIS_DSN), handle_client_shutdown=False)},
    on_startup=[pg_pool_startup, startup_fetch_missing_users],
    csrf_config=CSRFConfig(secret=CSRF_SECRET_TOKEN, cookie_name="s-csrf-token"),
    before_request=before_req,
    middleware=[session_auth_config.middleware],
    exception_handlers={
        HTTPException: plain_text_exception_handler,
        Exception: internal_error_handler,
    },
    debug=DEV,
)
