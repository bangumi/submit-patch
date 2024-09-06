### convert poetry.lock to requirements.txt ###
FROM python:3.10-slim@sha256:2407c61b1a18067393fecd8a22cf6fceede893b6aaca817bf9fbfe65e33614a3 AS poetry

WORKDIR /app

COPY requirements-poetry.txt ./
RUN --mount=type=cache,target=/root/.cache/pip pip install -r requirements-poetry.txt

COPY pyproject.toml poetry.lock ./
RUN --mount=type=cache,target=/root/.cache/pip poetry export -f requirements.txt --output requirements.txt --without-hashes

### final image ###
FROM python:3.10-slim@sha256:2407c61b1a18067393fecd8a22cf6fceede893b6aaca817bf9fbfe65e33614a3

WORKDIR /app

ENV PYTHONPATH=/app

COPY --from=poetry /app/requirements.txt ./requirements.txt

RUN --mount=type=cache,target=/root/.cache/pip pip install -U pip && \
    pip install -r requirements.txt

WORKDIR /app

ENTRYPOINT [ "uvicorn", "server.app:app" ]

COPY . ./
