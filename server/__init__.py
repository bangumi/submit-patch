import difflib
from datetime import datetime

import litestar
from litestar.config.csrf import CSRFConfig
from litestar.contrib.jinja import JinjaTemplateEngine
from litestar.exceptions import (
    NotAuthorizedException,
    NotFoundException,
)
from litestar.response import Template
from litestar.stores.redis import RedisStore
from litestar.template import TemplateConfig
from redis.asyncio.client import Redis

from config import (
    CSRF_SECRET_TOKEN,
    REDIS_DSN,
    UTC,
)
from server import tmpl
from server.auth import callback, login, session_auth_config
from server.base import Request, pg, pg_pool_startup
from server.contrib import suggest_api, suggest_ui
from server.model import Patch
from server.review import review_patch


@litestar.get("/")
async def index(request: Request) -> Template:
    if not request.auth:
        return Template("index-login.html")

    if not request.auth.allow_edit:
        rows = await pg.fetch(
            "select * from patch where from_user_id = $1 and deleted_at is NULL order by created_at desc",
            request.auth.user_id,
        )
        return Template("contrib/index.html.jinja2", context={"rows": rows})

    rows = await pg.fetch(
        "select * from patch where from_user_id = $1 and deleted_at is NULL order by created_at desc",
        request.auth.user_id,
    )

    return Template("wiki/index.html.jinja2", context={"rows": rows})


@litestar.get("/patches/{user_id:int}")
async def show_patches(user_id: int, request: Request) -> Template:
    rows = await pg.fetch(
        "select * from patch where from_user_id = $1 and deleted_at is NULL",
        request.auth.user_id,
    )
    return Template("contrib/index.html.jinja2", context={"rows": rows})


@litestar.get("/patch/{patch_id:str}")
async def get_patch(patch_id: str, request: Request) -> Template:
    p = await pg.fetchrow("""select * from patch where id = $1 and deleted_at is NULL""", patch_id)
    if not p:
        raise NotFoundException()

    patch = Patch(**p)

    name_patch = ""
    if patch.name is not None:
        name_patch = "".join(
            difflib.unified_diff([patch.original_name], [patch.name], "name", "name")
        )

    infobox_patch = ""
    if patch.infobox is not None:
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


@litestar.post("/api/delete-patch/{patch_id:str}")
async def delete_patch(patch_id: str, request: Request) -> Template:
    if not request.auth:
        raise NotAuthorizedException

    p = await pg.fetchrow("""select * from patch where id = $1 and deleted_at is NULL""", patch_id)
    if not p:
        raise NotFoundException()

    patch = Patch(**p)

    if patch.from_user_id != request.auth.user_id:
        raise NotAuthorizedException

    await pg.execute(
        "update patch set deleted_at = $1 where id = $2 ",
        datetime.now(tz=UTC),
        patch_id,
    )

    return Template("patch.html.jinja2", context={"patch": p, "auth": request.auth})


def before_req(req: litestar.Request):
    req.state["now"] = datetime.now(tz=UTC)


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
        show_patches,
    ],
    template_config=TemplateConfig(
        engine=JinjaTemplateEngine(engine_instance=tmpl.engine),
    ),
    stores={"sessions": RedisStore(Redis.from_url(REDIS_DSN), handle_client_shutdown=False)},
    on_startup=[pg_pool_startup],
    csrf_config=CSRFConfig(secret=CSRF_SECRET_TOKEN, cookie_name="csrf-token"),
    before_request=before_req,
    middleware=[session_auth_config.middleware],
    debug=True,
)
