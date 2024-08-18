import difflib
from datetime import datetime

import litestar
from litestar import Response
from litestar.config.csrf import CSRFConfig
from litestar.contrib.jinja import JinjaTemplateEngine
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
    REDIS_DSN,
    UTC,
)
from server import tmpl
from server.auth import callback, login, session_auth_config
from server.base import Request, pg, pg_pool_startup
from server.contrib import delete_patch, suggest_api, suggest_ui
from server.model import Patch
from server.review import review_patch


@litestar.get("/")
async def index(request: Request) -> Template:
    if not request.auth:
        return Template("login.html")

    if not request.auth.allow_edit:
        rows = await pg.fetch(
            "select * from patch where from_user_id = $1 and deleted_at is NULL order by created_at desc",
            request.auth.user_id,
        )
        return Template("index.html.jinja2", context={"rows": rows, "auth": request.auth})

    rows = await pg.fetch(
        "select * from patch where deleted_at is NULL order by created_at desc",
    )

    return Template("index.html.jinja2", context={"rows": rows, "auth": request.auth})


@litestar.get("/patch/{patch_id:str}")
async def get_patch(patch_id: str, request: Request) -> Template:
    p = await pg.fetchrow("""select * from patch where id = $1 and deleted_at is NULL""", patch_id)
    if not p:
        raise NotFoundException()

    patch = Patch(**p)

    name_patch = ""
    if patch.name is not None:
        if patch.original_name is None:
            logger.error("broken patch {!r}", patch_id)
            raise InternalServerException
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
                patch.original_summary.splitlines(True),
                patch.summary.splitlines(True),
                "summary",
                "summary",
            )
        )

    return Template(
        "patch.html.jinja2",
        context={
            "patch": p,
            "auth": request.auth,
            "name_patch": name_patch,
            "infobox_patch": infobox_patch,
            "summary_patch": summary_patch,
        },
    )


def before_req(req: litestar.Request):
    req.state["now"] = datetime.now(tz=UTC)


def plain_text_exception_handler(_: Request, exc: Exception) -> Response:
    """Default handler for exceptions subclassed from HTTPException."""
    status_code = getattr(exc, "status_code", HTTP_500_INTERNAL_SERVER_ERROR)
    detail = getattr(exc, "detail", "")

    return Template(
        "error.html.jinja2",
        status_code=status_code,
        context={
            "error": exc,
            "detail": detail,
        },
    )


app = litestar.Litestar(
    [
        index,
        login,
        callback,
        suggest_ui,
        suggest_api,
        get_patch,
        delete_patch,
        review_patch,
    ],
    template_config=TemplateConfig(
        engine=JinjaTemplateEngine.from_environment(tmpl.engine),
    ),
    stores={"sessions": RedisStore(Redis.from_url(REDIS_DSN), handle_client_shutdown=False)},
    on_startup=[pg_pool_startup],
    csrf_config=CSRFConfig(secret=CSRF_SECRET_TOKEN, cookie_name="s-csrf-token"),
    before_request=before_req,
    middleware=[session_auth_config.middleware],
    exception_handlers={
        HTTPException: plain_text_exception_handler,
    },
    debug=DEV,
)
