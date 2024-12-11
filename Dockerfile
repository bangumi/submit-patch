# syntax=docker/dockerfile:1@sha256:db1ff77fb637a5955317c7a3a62540196396d565f3dd5742e76dddbb6d75c4c5

### convert poetry.lock to requirements.txt ###
FROM python:3.10-slim@sha256:af6f1b19eae3400ea3a569ba92d4819a527be4662971d51bb798c923bba30a81 AS poetry

WORKDIR /app

ENV PIP_ROOT_USER_ACTION=ignore

COPY requirements-poetry.txt ./
RUN pip install -r requirements-poetry.txt

COPY pyproject.toml poetry.lock ./
RUN poetry export -f requirements.txt --output requirements.txt

### final image ###
FROM python:3.10-slim@sha256:af6f1b19eae3400ea3a569ba92d4819a527be4662971d51bb798c923bba30a81

WORKDIR /app

ENV PYTHONPATH=/app

COPY --from=poetry /app/requirements.txt ./requirements.txt

ENV PIP_ROOT_USER_ACTION=ignore

RUN pip install -U pip && \
    pip install -r requirements.txt

WORKDIR /app

ENTRYPOINT [ "uvicorn", "server.app:app" ]

COPY . ./
