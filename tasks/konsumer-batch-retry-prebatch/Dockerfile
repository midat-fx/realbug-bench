# Base commit of kafka-konsumer with every module pre-fetched: grading runs with --network none.
FROM golang:1.25.14-bookworm@sha256:3b4a11519ad929d1e1d261a12cff056f0c85b735253d7d861346b9c6f8b36437
ENV GOTOOLCHAIN=local GOFLAGS=-buildvcs=false
RUN git clone --quiet https://github.com/Trendyol/kafka-konsumer /work/repo \
 && git -C /work/repo checkout --quiet d80db5c55df7fa8c83e1da30b5bbae10c9c9720a
WORKDIR /work/repo
RUN go mod download \
 && go build ./... \
 && go test -count=1 -run '^$' . >/dev/null \
 && go test -race -count=1 -run '^$' . >/dev/null
