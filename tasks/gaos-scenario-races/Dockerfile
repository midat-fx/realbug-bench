# Base commit of gaos with every module pre-fetched: grading runs with --network none.
FROM golang:1.25.14-bookworm@sha256:3b4a11519ad929d1e1d261a12cff056f0c85b735253d7d861346b9c6f8b36437
ENV GOTOOLCHAIN=local GOFLAGS=-buildvcs=false
RUN git clone --quiet https://github.com/Trendyol/gaos /work/repo \
 && git -C /work/repo checkout --quiet 771d620499118e64e048de918103bcc404bda74f
WORKDIR /work/repo
RUN go mod download \
 && go build ./... \
 && go test -count=1 -run '^$' ./... >/dev/null \
 && go test -race -count=1 -run '^$' ./runner/ >/dev/null
