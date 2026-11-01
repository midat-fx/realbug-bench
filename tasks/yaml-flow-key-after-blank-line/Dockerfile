# eemeli/yaml at the parent of the fix commit, with the yaml-test-suite and JSONTestSuite submodules;
# dependencies from the repository's package-lock.json.
FROM node:22-bookworm-slim@sha256:4b150b48b4116e2a22f2c39cc27025fbf4f153e2078fd3b14d159fdb5c7279da
RUN apt-get update -qq && apt-get install -y -qq --no-install-recommends git ca-certificates >/dev/null \
 && rm -rf /var/lib/apt/lists/*
ENV CI=true NO_COLOR=1 npm_config_update_notifier=false
RUN git clone --quiet https://github.com/eemeli/yaml /work/repo \
 && git -C /work/repo checkout --quiet 4ecbf764b11605e8e62ee3596f18b0a4f0f78c88 \
 && git -C /work/repo submodule update --quiet --init --depth 1 tests/yaml-test-suite tests/json-test-suite
WORKDIR /work/repo
RUN npm ci --ignore-scripts --no-audit --no-fund --loglevel=error \
 && npx tsc --noEmit && npx vitest run >/dev/null
