FROM ghcr.io/astral-sh/uv:debian-slim@sha256:2de381791ffdaff9397a47fba5507728185fb996a327aca19eabde4cdb269921 AS build

WORKDIR /app

COPY uv.lock pyproject.toml ./

RUN uv export --no-group dev --frozen --no-emit-project > /app/requirements.txt

FROM python:3.10-slim@sha256:66aad90b231f011cb80e1966e03526a7175f0586724981969b23903abac19081

ENV PIP_ROOT_USER_ACTION=ignore
WORKDIR /app

COPY --from=build /app/requirements.txt .

RUN pip install --only-binary=:all: --no-cache --no-deps -r requirements.txt

ENTRYPOINT [ "uvicorn", "server.app:app" ]

COPY . .
