FROM golang:1.25-alpine AS builder

WORKDIR /src/scm

RUN apk add --no-cache build-base

COPY scm/go.mod scm/go.sum ./
RUN go mod download

COPY scm/. .
COPY reference/seeder/parts.json /src/reference/seeder/parts.json

ENV CGO_ENABLED=1

RUN go build -trimpath -ldflags="-s -w" -o /out/server ./cmd/server

FROM alpine:3.22

RUN apk add --no-cache ca-certificates sqlite-libs

RUN addgroup -S -g 1001 scm && adduser -S -u 1001 -G scm scm

WORKDIR /app

ARG VERSION=dev

LABEL org.opencontainers.image.version=$VERSION

COPY --from=builder /out/server /app/server
COPY --from=builder /src/scm/docs ./docs

RUN chown -R scm:scm /app

USER scm

RUN mkdir -p /app/contracts

EXPOSE 8081

CMD ["./server"]