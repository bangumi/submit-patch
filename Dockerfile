FROM ghcr.io/astral-sh/uv:debian-slim@sha256:53abdd1f70b18b6a2e3212faa2c82bf79d0707c2facc0421c81187cfa071b7bb AS build

WORKDIR /app

COPY uv.lock pyproject.toml ./

RUN uv export --no-group dev --frozen --no-emit-project > /app/requirements.txt

FROM python:3.10-slim@sha256:06f6d69d229bb55fab83dded514e54eede977e33e92d855ba3f97ce0e3234abc

ENV PIP_ROOT_USER_ACTION=ignore
WORKDIR /app

COPY --from=build /app/requirements.txt .

RUN pip install --only-binary=:all: --no-cache --no-deps -r requirements.txt

ENTRYPOINT [ "uvicorn", "server.app:app" ]

COPY . .
