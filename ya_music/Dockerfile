FROM python:3.12-bookworm

WORKDIR /app

RUN apt-get update && apt-get install -y --no-install-recommends git curl
COPY --from=ghcr.io/astral-sh/uv:latest /uv /uvx /bin/

ENV PYTHONPATH=/app

COPY pyproject.toml /app/
COPY uv.lock /app/

RUN uv sync

COPY . /app/

