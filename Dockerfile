### convert poetry.lock to requirements.txt ###
FROM python:3.10-slim@sha256:8666a639a54acc810408e505e2c6b46b50834385701675ee177f578b3d2fdef9 AS poetry

WORKDIR /app

COPY requirements-poetry.txt ./
RUN --mount=type=cache,target=/root/.cache/pip pip install -r requirements-poetry.txt

COPY pyproject.toml poetry.lock ./
RUN --mount=type=cache,target=/root/.cache/pip poetry export -f requirements.txt --output requirements.txt --without-hashes

### final image ###
FROM python:3.10-slim@sha256:8666a639a54acc810408e505e2c6b46b50834385701675ee177f578b3d2fdef9

WORKDIR /app

ENV PYTHONPATH=/app

COPY --from=poetry /app/requirements.txt ./requirements.txt

RUN --mount=type=cache,target=/root/.cache/pip pip install -U pip && \
    pip install -r requirements.txt

WORKDIR /app

ENTRYPOINT [ "uvicorn", "server.app:app" ]

COPY . ./
