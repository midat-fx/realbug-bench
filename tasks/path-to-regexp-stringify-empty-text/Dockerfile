# path-to-regexp at the parent of the fix commit. The repository has no lockfile, so dependencies are
# resolved as of the base commit's date (npm --before) to keep the image reproducible.
FROM node:22-bookworm-slim@sha256:4b150b48b4116e2a22f2c39cc27025fbf4f153e2078fd3b14d159fdb5c7279da
RUN apt-get update -qq && apt-get install -y -qq --no-install-recommends git ca-certificates >/dev/null \
 && rm -rf /var/lib/apt/lists/*
ENV CI=true NO_COLOR=1 npm_config_update_notifier=false
RUN git clone --quiet https://github.com/pillarjs/path-to-regexp /work/repo \
 && git -C /work/repo checkout --quiet 8877f41873e37a30258d3935feaf1d2679321735
WORKDIR /work/repo
RUN npm install --before=2026-08-29T00:00:00Z --ignore-scripts --no-audit --no-fund --loglevel=error \
 && npx tsc --noEmit -p tsconfig.json && npx vitest run src/index.spec.ts >/dev/null
