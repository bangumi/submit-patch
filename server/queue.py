import asyncio

from bgm_tv_wiki import WikiSyntaxError, parse
from litestar import Litestar
from loguru import logger
from typing_extensions import Never
from uuid_utils import uuid7

from server.base import pg, subject_infobox_queue
from server.model import PatchType


async def on_app_start_queue(app: Litestar) -> None:
    async def queue_handler() -> Never:
        logger.info("start queue handler")
        while True:
            item = await subject_infobox_queue.get()
            with logger.catch():
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

                    await pg.execute(
                        """
                        insert into edit_suggestion (id, patch_id, patch_type, text, from_user)
                        VALUES ($1, $2, $3, $4, $5)
                    """,
                        uuid7(),
                        item.patch_id,
                        PatchType.Subject,
                        "infobox 包含语法错误，请检查\n" + msg,
                        287622,
                    )

    # keep a ref so task won't be GC-ed.
    app.state["background_queue_handler"] = asyncio.create_task(queue_handler())
