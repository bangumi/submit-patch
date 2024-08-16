from typing import Any

import httpx

from config import BGM_TV_TOKEN


class Error(Exception):
    code: str
    error: str
    message: str

    def __init__(self, code: str, error: str, message: str):
        super().__init__(f"[{code}]: {message}")
        self.code = code
        self.error = error
        self.message = message


client = httpx.Client(headers={"x-api-key": f"Bearer {BGM_TV_TOKEN}"})


def __strip_none(d: dict[str, Any]) -> dict[str, Any]:
    return {key: value for key, value in d.items() if value is not None}


def edit_subject(
    subject_id: int,
    commit_message: str,
    *,
    infobox: str | None = None,
    name: str | None = None,
    summary: str | None = None,
    nsfw: bool | None = None,
):
    r = client.patch(
        f"https://next.bgm.tv/p1/wiki/subjects/{subject_id}",
        json={
            "commitMessage": commit_message,
            "subject": __strip_none(
                {
                    "infobox": infobox,
                    "name": name,
                    "summary": summary,
                    "nsfw": nsfw,
                }
            ),
        },
    )

    if r.status_code >= 400:
        data = r.json()
        raise Error(code=data["code"], error=data["error"], message=data["message"])


if __name__ == "__main__":
    edit_subject(
        363612,
        "test-api",
        summary="""
本条目是一个沙盒，可以用于尝试bgm功能。

普通维基人可以随意编辑条目信息以及相关关联查看编辑效果，但是请不要完全删除沙盒说明并且不要关联非沙盒条目/人物/角色。

https://bgm.tv/group/topic/366812#post_1923517

ss
""",
    )
