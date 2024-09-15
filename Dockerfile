### convert poetry.lock to requirements.txt ###
FROM python:3.10-slim@sha256:80619a5316afae7045a3c13371b0ee670f39bac46ea1ed35081d2bf91d6c3dbd AS poetry

WORKDIR /app

COPY requirements-poetry.txt ./
RUN --mount=type=cache,target=/root/.cache/pip pip install -r requirements-poetry.txt

COPY pyproject.toml poetry.lock ./
RUN --mount=type=cache,target=/root/.cache/pip poetry export -f requirements.txt --output requirements.txt --without-hashes

### final image ###
FROM python:3.10-slim@sha256:80619a5316afae7045a3c13371b0ee670f39bac46ea1ed35081d2bf91d6c3dbd

WORKDIR /app

ENV PYTHONPATH=/app

COPY --from=poetry /app/requirements.txt ./requirements.txt

RUN --mount=type=cache,target=/root/.cache/pip pip install -U pip && \
    pip install -r requirements.txt

WORKDIR /app

ENTRYPOINT [ "uvicorn", "server.app:app" ]

COPY . ./
