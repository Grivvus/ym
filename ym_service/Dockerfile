FROM python:3.13-slim-bookworm

WORKDIR /app

RUN apt-get update && apt-get install -y --no-install-recommends git curl
COPY --from=ghcr.io/astral-sh/uv:latest /uv /uvx /bin/

ENV PYTHONPATH=/app

COPY pyproject.toml /app/

RUN uv sync --no-dev

COPY . /app/

CMD ["uv", "run", "--no-dev", "/app/app/main.py"]
