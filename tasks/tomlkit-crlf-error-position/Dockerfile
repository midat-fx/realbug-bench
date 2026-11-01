# tomlkit at the parent of the fix commit, with the toml-test submodule. Pure Python; pytest and PyYAML
# (used by the regression suite) are pinned.
FROM python:3.12-slim-bookworm@sha256:e697c81897302a4c4333bade000d98038fbb21eb064f8ee1fd42d02ebcd9d337
RUN apt-get update -qq && apt-get install -y -qq --no-install-recommends git ca-certificates >/dev/null \
 && rm -rf /var/lib/apt/lists/*
ENV PIP_DISABLE_PIP_VERSION_CHECK=1 PYTHONDONTWRITEBYTECODE=1
RUN pip install -q --no-cache-dir pytest==8.4.2 PyYAML==6.0.3
RUN git clone --quiet https://github.com/python-poetry/tomlkit /work/repo \
 && git -C /work/repo checkout --quiet 8c30fc17d91aee7469c0287e509300e78aa15227 \
 && git -C /work/repo submodule update --quiet --init --depth 1 tests/toml-test
WORKDIR /work/repo
RUN python -m pytest -q -p no:cacheprovider tests >/dev/null
