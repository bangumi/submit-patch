FROM ghcr.io/astral-sh/uv:debian-slim@sha256:9c5263a75e4d491f7df38996dbfadde45aa8dfe11e0b0f484cdd3fe43cb85bf5 AS build

WORKDIR /app

COPY uv.lock pyproject.toml ./

RUN uv export --no-group dev --frozen --no-emit-project > /app/requirements.txt

FROM python:3.10-slim@sha256:f9fd9a142c9e3bc54d906053b756eb7e7e386ee1cf784d82c251cf640c502512

ENV PIP_ROOT_USER_ACTION=ignore
WORKDIR /app

COPY --from=build /app/requirements.txt .

RUN pip install --only-binary=:all: --no-cache --no-deps -r requirements.txt

ENTRYPOINT [ "uvicorn", "server.app:app" ]

COPY . .
