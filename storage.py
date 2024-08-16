# ruff: noqa: N803
from contextlib import AbstractContextManager
from typing import Any, Protocol, cast

import botocore.session
from psycopg import Connection as _Connection
from psycopg import Cursor
from psycopg.rows import dict_row
from psycopg_pool import ConnectionPool

from config import S3_ACCESS_KEY, S3_SECRET_KEY


class S3Client(Protocol):
    def head_object(self, *, Bucket: str, Key: str) -> Any: ...

    def put_object(self, *, Bucket: str, Key: str, Body: bytes, ContentType: str | None = None): ...


s3_client: S3Client = cast(
    S3Client,
    botocore.session.get_session().create_client(
        "s3",
        aws_access_key_id=S3_ACCESS_KEY,
        aws_secret_access_key=S3_SECRET_KEY,
        endpoint_url="https://static.trim21.cn/",
    ),
)


class Connection(_Connection):
    def run(self, sql, params=None):
        with self.cursor() as cur:
            cur.execute(query=sql, params=params)
        self.commit()

    def dict_cursor(self) -> AbstractContextManager[Cursor[dict]]:
        return self.cursor(row_factory=dict_row)

    def fetch_one(
        self, query, params=None, *, prepare: bool | None = None
    ) -> tuple[Any, ...] | None:
        with self.cursor() as cur:
            return cur.execute(query, params, prepare=prepare).fetchone()

    def fetch_val(self, query, params=None, *, prepare: bool | None = None) -> Any:
        with self.cursor() as cur:
            v = cur.execute(query, params, prepare=prepare).fetchone()
            if v:
                return v[0]

    def query_dict_one(
        self, query, params=None, *, prepare: bool | None = None
    ) -> dict[str, Any] | None:
        with self.cursor(row_factory=dict_row) as cur:
            return cur.execute(query, params, prepare=prepare).fetchone()

    def must_query_dict_one(
        self, query, params=None, *, prepare: bool | None = None
    ) -> dict[str, Any]:
        with self.cursor(row_factory=dict_row) as cur:
            v = cur.execute(query, params, prepare=prepare).fetchone()
        if v is None:
            raise ValueError("record not found")
        return v

    def query_dict(
        self, query, params: tuple | list | None = None, *, prepare: bool | None = None
    ) -> list[dict[str, Any]]:
        with self.cursor(row_factory=dict_row) as cur:
            rr = cur.execute(query, params, prepare=prepare).fetchall()
            return rr

    def query(self, query, params=None, *, prepare: bool | None = None) -> list[tuple[Any, ...]]:
        with self.cursor() as cur:
            return cur.execute(query, params, prepare=prepare).fetchall()


class _Pool(ConnectionPool[Connection]):
    def connection(self, timeout: float | None = None) -> AbstractContextManager[Connection]:  # type: ignore[override]
        return super().connection()


pg_pool = _Pool(
    conninfo="user=postgres password=postgres host=192.168.1.3 port=5433 dbname=pt",
    connection_class=Connection,
    max_size=20,
)
