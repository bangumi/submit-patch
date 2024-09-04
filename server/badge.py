import asyncio

import litestar
from litestar import Response

from server.base import (
    disable_cookies_opt,
    http_client,
    pg,
    redis_client,
)
from server.model import PatchState
from server.router import Router


router = Router()

__cache_headers = {"Cache-Control": "public, max-age=5"}


@router
@litestar.get(
    "/badge/subject/{subject_id:int}",
    response_headers=__cache_headers,
    opt=disable_cookies_opt,
)
async def badge_handle_subject(subject_id: int) -> Response[bytes]:
    count = sum(
        await asyncio.gather(
            pg.fetchval(
                "select count(1) from view_subject_patch where state = $1 AND subject_id=$2",
                PatchState.Pending,
                subject_id,
            ),
            pg.fetchval(
                "select count(1) from view_episode_patch where state = $1 AND subject_id=$2",
                PatchState.Pending,
                subject_id,
            ),
        )
    )

    badge = await __get_badge(count)
    return Response(badge, media_type="image/svg+xml")


@router
@litestar.get(
    "/badge/episode/{episode_id:int}",
    response_headers=__cache_headers,
    opt=disable_cookies_opt,
)
async def badge_handle_episode(episode_id: int) -> Response[bytes]:
    count = await pg.fetchval(
        "select count(1) from view_episode_patch where state = $1 AND episode_id=$2",
        PatchState.Pending,
        episode_id,
    )

    badge = await __get_badge(count)
    return Response(badge, media_type="image/svg+xml")


@router
@litestar.get("/badge.svg", response_headers=__cache_headers, opt=disable_cookies_opt)
async def badge_handle() -> Response[bytes]:
    key = "patch:rest:pending"
    pending = await redis_client.get(key)

    if pending is not None:
        return Response(pending, media_type="image/svg+xml")

    rest = sum(
        await asyncio.gather(
            pg.fetchval(
                "select count(1) from view_subject_patch where state = $1",
                PatchState.Pending,
            ),
            pg.fetchval(
                "select count(1) from view_episode_patch where state = $1",
                PatchState.Pending,
            ),
        )
    )

    badge = await __get_badge(rest)

    await redis_client.set(key, badge, ex=10)

    return Response(badge, media_type="image/svg+xml")


async def __get_badge(count: int) -> bytes:
    key = "badge:count"

    if count >= 100:
        val_key = f"{key}:{count // 100 * 100}"
    else:
        val_key = f"{key}:{count}"

    badge = await redis_client.get(val_key)

    s = str(count)

    if badge is None:
        if count >= 100:
            s = f">{count // 100 * 100}"
            color = "dc3545"
        elif count >= 50:
            color = "ffc107"
        else:
            color = "green"

        res = await http_client.get(f"https://img.shields.io/badge/待审核-{s}-{color}")
        badge = res.content
        await redis_client.set(val_key, badge, ex=7 * 24 * 3600)

    return badge
