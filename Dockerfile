### convert poetry.lock to requirements.txt ###
FROM python:3.12-slim AS poetry

WORKDIR /app

COPY requirements-poetry.txt ./
RUN --mount=type=cache,target=/root/.cache/pip pip install -r requirements-poetry.txt

COPY pyproject.toml poetry.lock ./
RUN --mount=type=cache,target=/root/.cache/pip poetry export -f requirements.txt --output requirements.txt --without-hashes

### final image ###
FROM python:3.12-slim

WORKDIR /app

ENV PYTHONPATH=/app

COPY --from=poetry /app/requirements.txt ./requirements.txt

RUN --mount=type=cache,target=/root/.cache/pip pip install -U pip && \
    pip install -r requirements.txt

WORKDIR /app

ENTRYPOINT [ "uvicorn", "server.app:app" ]

COPY . ./
