import html
from urllib.parse import urlparse

import httpx
from pg8000.native import Connection

from config import PG_DSN


u = urlparse(PG_DSN)


def main():
    c = httpx.Client()
    conn = Connection(
        host=u.hostname,
        user=u.username,
        port=u.port,
        password=u.password,
        database=u.path.removeprefix("/"),
    )

    results: list[tuple[int, int]] = conn.run("select from_user_id, wiki_user_id from patch")
    s = set()
    for u1, u2 in results:
        s.add(u1)
        s.add(u2)

    s.discard(0)

    for user in s:
        r = c.get(f"https://api.bgm.tv/user/{user}")
        data = r.json()
        conn.run(
            """
        insert into patch_users (user_id, username, nickname) VALUES (:id,:username,:nickname)
        on conflict (user_id) do update set
            username = excluded.username,
            nickname = excluded.nickname
    """,
            id=data["id"],
            username=data["username"],
            nickname=html.unescape(data["nickname"]),
        )


if __name__ == "__main__":
    main()
