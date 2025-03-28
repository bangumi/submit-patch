import asyncio
import html
import logging
import mimetypes
import os
import re
import uuid
from datetime import datetime, timedelta, timezone
from pathlib import Path
from typing import Any, NamedTuple

import litestar
import sslog
from litestar import Response
from litestar.config.csrf import CSRFConfig
from litestar.contrib.jinja import JinjaTemplateEngine
from litestar.datastructures import State
from litestar.exceptions import (
    HTTPException,
    NotFoundException,
)
from litestar.response import Redirect, Template
from litestar.static_files import create_static_files_router
from litestar.status_codes import HTTP_500_INTERNAL_SERVER_ERROR
from litestar.stores.redis import RedisStore
from litestar.template import TemplateConfig
from sslog import logger

from server import auth, badge, contrib, index, patch, review, tmpl
from server.auth import require_user_login, session_auth_config
from server.base import (
    AuthorizedRequest,
    RedirectException,
    Request,
    User,
    XRequestIdMiddleware,
    external_http_client,
    pg,
    pg_pool_startup,
    redis_client,
)
from server.config import (
    CSRF_SECRET_TOKEN,
    DEV,
    PROJECT_PATH,
    UTC,
)
from server.migration import run_migration
from server.queue import on_app_start_queue
from server.router import Router


httpx_logger = logging.getLogger("httpx")
for h in httpx_logger.handlers:
    httpx_logger.removeHandler(h)

httpx_logger.propagate = False
httpx_logger.addHandler(sslog.InterceptHandler())


router = Router()


class File(NamedTuple):
    content: bytes
    content_type: str | None


mimetypes.add_type("application/javascript", ".mjs", True)

static_path = PROJECT_PATH.joinpath("static/")
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
                headers={"cache-control": "public, max-age=604800"},  # a week
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

# \d will match many Unicode number
__tz_pattern = re.compile(r"-?[0-9]+")


def before_req(req: litestar.Request[None, User | None, State]) -> None:
    req.state["now"] = datetime.now(tz=UTC)

    a = req.scope.get("auth")
    if a is not None:
        req.state["tz"] = timezone(timedelta(hours=int(a.time_offset)))
        return

    tz = req.cookies.get("tz")
    if tz:
        if __tz_pattern.match(tz):
            req.state["tz"] = timezone(timedelta(minutes=-int(tz)))


class MediaType:
    html = "text/html"
    json = "application/json"


def plain_text_exception_handler(req: Request, exc: HTTPException) -> Response[Any]:
    """Default handler for exceptions subclassed from HTTPException."""
    accept = req.accept.best_match([MediaType.html, MediaType.json])
    if accept == MediaType.json:
        return Response(
            content={
                "error": type(exc).__name__,
                "method": req.method,
                "url": str(req.url),
                "extra": exc.extra,
                "detail": exc.detail,
            },
            status_code=exc.status_code,
        )

    return Template(
        "error.html.jinja2",
        status_code=exc.status_code,
        context={
            "error": repr(exc),
            "method": req.method,
            "url": str(req.url),
            "extra": exc.extra,
            "detail": exc.detail,
        },
    )


def internal_error_handler(req: Request, exc: Exception) -> Response[Any]:
    logger.exception(f"internal server error: {type(exc)} {exc}")

    return Response(
        content={
            "status_code": 500,
            "detail": "Internal Server Error",
            "method": req.method,
            "url": str(req.url),
        },
        status_code=HTTP_500_INTERNAL_SERVER_ERROR,
    )


async def startup_fetch_missing_users() -> None:
    logger.info("fetch missing users")
    s: set[int | None] = set()

    for u1, u2 in await pg.fetch("select from_user_id, wiki_user_id from subject_patch"):
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
        r = await external_http_client.get(f"https://api.bgm.tv/user/{user}")
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


async def refresh_db(application: litestar.Litestar) -> None:
    async def refresh() -> None:
        last_patch_id = uuid.UUID(int=0)
        while True:
            rows = await pg.fetch(
                "select id from subject_patch where id > $1 order by id limit 20", last_patch_id
            )
            if not rows:
                break
            for (patch_id,) in rows:
                last_patch_id = patch_id
                await pg.execute(
                    """
                update subject_patch
                    set comments_count = (
                        select count(1)
                        from edit_suggestion
                        where patch_type = 'subject' and patch_id = $1
                    )
                where id = $1
                """,
                    patch_id,
                )

    # keep a ref so task won't be GC-ed.
    application.state["background_refresh-db"] = asyncio.create_task(refresh())


def exception_as_redirect(req: Request, exc: RedirectException) -> Response[Any]:
    return Redirect(exc.location)


@router
@litestar.get("/api/debug", guards=[require_user_login])
async def debug_router(request: AuthorizedRequest) -> Any:
    if request.auth.super_user():
        return request.session

    return {}


app = litestar.Litestar(
    [
        *index.router,
        *auth.router,
        *contrib.router,
        *review.router,
        *patch.router,
        *badge.router,
        *router,
    ],
    template_config=TemplateConfig(
        engine=JinjaTemplateEngine.from_environment(tmpl.engine),
    ),
    stores={"sessions": RedisStore(redis_client, handle_client_shutdown=False)},
    on_startup=[
        pg_pool_startup,
        run_migration,
        startup_fetch_missing_users,
        on_app_start_queue,
        refresh_db,
    ],
    csrf_config=CSRFConfig(secret=CSRF_SECRET_TOKEN, cookie_name="s-csrf-token"),
    before_request=before_req,
    middleware=[XRequestIdMiddleware, session_auth_config.middleware],
    exception_handlers={
        RedirectException: exception_as_redirect,
        HTTPException: plain_text_exception_handler,
        Exception: internal_error_handler,
    },
    debug=DEV,
)
