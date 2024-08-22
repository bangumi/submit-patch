### convert poetry.lock to requirements.txt ###
FROM python:3.12-slim AS poetry

WORKDIR /app
COPY . ./
COPY pyproject.toml poetry.lock requirements-poetry.txt ./

RUN pip install -r requirements-poetry.txt &&\
  poetry export -f requirements.txt --output requirements.txt --without-hashes

### final image ###
FROM python:3.12-slim

WORKDIR /app

ENV PYTHONPATH=/app

COPY --from=poetry /app/requirements.txt ./requirements.txt

RUN pip install -U pip && \
    pip install -r requirements.txt --no-cache-dir

WORKDIR /app

ENTRYPOINT [ "uvicorn", "server:app" ]

COPY . ./
