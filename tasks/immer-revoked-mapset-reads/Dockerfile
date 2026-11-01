# immer at the parent of the fix commit; dependencies from the repository's yarn.lock (Yarn 1, bundled
# with the node image).
FROM node:22-bookworm-slim@sha256:4b150b48b4116e2a22f2c39cc27025fbf4f153e2078fd3b14d159fdb5c7279da
RUN apt-get update -qq && apt-get install -y -qq --no-install-recommends git ca-certificates >/dev/null \
 && rm -rf /var/lib/apt/lists/*
ENV CI=true NO_COLOR=1 npm_config_update_notifier=false
RUN git clone --quiet https://github.com/immerjs/immer /work/repo \
 && git -C /work/repo checkout --quiet 02e87e7b96d0e12ba223203cc66e8fad4049c482
WORKDIR /work/repo
RUN yarn install --frozen-lockfile --ignore-scripts --non-interactive --silent \
 && yarn cache clean --silent \
 && npx tsc --noEmit -p tsconfig.json --skipLibCheck && npx vitest run >/dev/null
