import asyncio

from bgm_tv_wiki import WikiSyntaxError, parse
from litestar import Litestar
from sslog import logger

from server.base import QueueItem, pg, subject_infobox_queue
from server.db import create_edit_suggestion
from server.model import PatchType


async def check_infobox_error(item: QueueItem) -> None:
    try:
        parse(item.infobox)
    except WikiSyntaxError as e:
        if e.lino:
            msg = f"line {e.lino}\n"
        else:
            msg = ""

        if e.lino:
            msg = msg + f"{e.line}\n"

        msg = msg + e.message

        await create_edit_suggestion(
            pg,
            item.patch_id,
            PatchType.Subject,
            "infobox 包含维基语法错误，请查看 https://bgm.tv/group/topic/365856 了解具体详情 \n"
            + msg,
            287622,
        )
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
            item.patch_id,
        )


async def on_app_start_queue(app: Litestar) -> None:
    async def queue_handler() -> None:
        logger.info("start queue handler")
        while True:
            item = await subject_infobox_queue.get()
            try:
                await check_infobox_error(item)
            except Exception as e:  # noqa: BLE001
                logger.exception(f"queue error {e!r}")

    # keep a ref so task won't be GC-ed.
    app.state["background_queue_handler"] = asyncio.create_task(queue_handler())
