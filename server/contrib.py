from dataclasses import dataclass
from typing import Annotated

import litestar
from litestar.enums import RequestEncodingType
from litestar.exceptions import PermissionDeniedException, ValidationException
from litestar.params import Body
from litestar.response import Template

from server.base import Request, http_client, pg
from server.model import Wiki


@litestar.get("/suggest")
async def suggest_ui(subject_id: int = 0) -> Template:
    if subject_id == 0:
        return Template("select-subject.html.jinja2")
    async with http_client.get(f"https://next.bgm.tv/p1/wiki/subjects/{subject_id}") as res:
        data = await res.json()
    return Template("suggest.html.jinja2", context={"data": data, "subject_id": subject_id})


@dataclass(frozen=True, slots=True)
class CreateSuggestion:
    name: str
    infobox: str
    summary: str
    desc: str
    nsfw: bool = False


@litestar.post("/suggest")
async def suggest_api(
    subject_id: int,
    data: Annotated[CreateSuggestion, Body(media_type=RequestEncodingType.URL_ENCODED)],
    request: Request,
) -> CreateSuggestion:
    if not request.auth:
        raise PermissionDeniedException
    if request.auth.allow_edit:
        raise PermissionDeniedException

    if not data.desc:
        raise ValidationException("missing suggestion description")

    async with http_client.get(f"https://next.bgm.tv/p1/wiki/subjects/{subject_id}") as res:
        res.raise_for_status()
        original_wiki = await res.json()

    original = Wiki(
        name=original_wiki["name"],
        infobox=original_wiki["infobox"],
        summary=original_wiki["summary"],
        nsfw=original_wiki["nsfw"],
    )

    name: str | None = None
    summary: str | None = None
    infobox: str | None = None

    original_name: str | None = None
    original_summary: str | None = None
    original_infobox: str | None = None

    nsfw: bool | None = None

    if original.name != data.name:
        name = data.name
        original_name = original.name

    if original.infobox != data.infobox:
        infobox = data.infobox
        original_infobox = original.infobox

    if original.summary != data.summary:
        summary = data.summary
        original_summary = original.summary

    if original.nsfw != data.nsfw:
        nsfw = data.nsfw

    await pg.execute(
        """
        insert into patch (subject_id, from_user_id, description, name, infobox, summary, nsfw,original_name,original_infobox,original_summary)
        VALUES ($1, $2, $3, $4, $5, $6, $7,$8,$9,$10)
    """,
        subject_id,
        request.auth.user_id,
        data.desc,
        name,
        infobox,
        summary,
        nsfw,
        original_name,
        original_infobox,
        original_summary,
    )

    return data
