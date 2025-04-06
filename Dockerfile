FROM ghcr.io/astral-sh/uv:debian-slim@sha256:3c3ca15d7011789f6bd703acc8b8c2533da4ff94ce2d8281bf0420eb33db891f AS build

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
