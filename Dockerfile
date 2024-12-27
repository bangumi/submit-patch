# syntax=docker/dockerfile:1@sha256:93bfd3b68c109427185cd78b4779fc82b484b0b7618e36d0f104d4d801e66d25

### convert poetry.lock to requirements.txt ###
FROM python:3.10-slim@sha256:bdc6c5b8f725df8b009b32da65cbf46bfd24d1c86dce2e6169452c193ad660b4 AS poetry

WORKDIR /app

ENV PIP_ROOT_USER_ACTION=ignore

COPY requirements-poetry.txt ./
RUN pip install -r requirements-poetry.txt

COPY pyproject.toml poetry.lock ./
RUN poetry export -f requirements.txt --output requirements.txt

### final image ###
FROM python:3.10-slim@sha256:bdc6c5b8f725df8b009b32da65cbf46bfd24d1c86dce2e6169452c193ad660b4

WORKDIR /app

ENV PYTHONPATH=/app

COPY --from=poetry /app/requirements.txt ./requirements.txt

ENV PIP_ROOT_USER_ACTION=ignore

RUN pip install -U pip && \
    pip install -r requirements.txt

WORKDIR /app

ENTRYPOINT [ "uvicorn", "server.app:app" ]

COPY . ./
