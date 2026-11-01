# sqlparse at the parent of the fix commit. Pure Python, no runtime dependencies; pytest is pinned.
FROM python:3.12-slim-bookworm@sha256:e697c81897302a4c4333bade000d98038fbb21eb064f8ee1fd42d02ebcd9d337
RUN apt-get update -qq && apt-get install -y -qq --no-install-recommends git ca-certificates >/dev/null \
 && rm -rf /var/lib/apt/lists/*
ENV PIP_DISABLE_PIP_VERSION_CHECK=1 PYTHONDONTWRITEBYTECODE=1
RUN pip install -q --no-cache-dir pytest==8.4.2
RUN git clone --quiet https://github.com/andialbrecht/sqlparse /work/repo \
 && git -C /work/repo checkout --quiet df8e28495a87954a9d52de70c33ef7225c53b28f
WORKDIR /work/repo
RUN python -m pytest -q -p no:cacheprovider tests >/dev/null
