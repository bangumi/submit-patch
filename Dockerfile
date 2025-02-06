FROM ghcr.io/astral-sh/uv:debian-slim AS build

WORKDIR /app

COPY uv.lock pyproject.toml ./

RUN uv export --no-group dev --frozen --no-emit-project > /app/requirements.txt

FROM python:3.10-slim

RUN apt-get update && apt-get install -y \
    ffmpeg \
    mediainfo &&\
    rm -rf /var/lib/apt/lists/*

WORKDIR /app

COPY --from=build /app/requirements.txt .

ENV PIP_ROOT_USER_ACTION=ignore

RUN pip install --only-binary=:all: --no-cache -r requirements.txt

# check oxipng is working
RUN oxipng --version

ENTRYPOINT [ "uvicorn", "server.app:app" ]

COPY . .
