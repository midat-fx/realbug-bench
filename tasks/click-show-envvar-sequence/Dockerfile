# click at the parent of the fix commit. Pure Python, no runtime dependencies; installed editable
# so the candidate's changes under src/ are what the tests import. pytest is pinned; less is for the
# pager tests in the suite.
FROM python:3.12-slim-bookworm@sha256:e697c81897302a4c4333bade000d98038fbb21eb064f8ee1fd42d02ebcd9d337
RUN apt-get update -qq && apt-get install -y -qq --no-install-recommends git ca-certificates less >/dev/null \
 && rm -rf /var/lib/apt/lists/*
ENV PIP_DISABLE_PIP_VERSION_CHECK=1 PYTHONDONTWRITEBYTECODE=1
RUN pip install -q --no-cache-dir pytest==8.4.2
RUN git clone --quiet https://github.com/pallets/click /work/repo \
 && git -C /work/repo checkout --quiet bb695aaecdef95832d167fc1d508921745aa824e
WORKDIR /work/repo
RUN pip install -q --no-cache-dir --no-deps -e . && python -m pytest -q -p no:cacheprovider tests >/dev/null
