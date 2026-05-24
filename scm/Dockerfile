FROM golang:1.25-alpine AS builder

WORKDIR /app

RUN apk add --no-cache build-base

COPY go.mod go.sum ./
RUN go mod download

COPY . .

ENV CGO_ENABLED=1

RUN go build -trimpath -ldflags="-s -w" -o /out/server ./cmd/server && \
	go build -trimpath -ldflags="-s -w" -o /out/migrate ./cmd/migrate && \
	go build -trimpath -ldflags="-s -w" -o /out/seeder ./cmd/seeder

FROM alpine:3.22

RUN apk add --no-cache ca-certificates sqlite-libs

WORKDIR /app

ARG VERSION=dev

LABEL org.opencontainers.image.version=$VERSION

COPY --from=builder /out/server /app/server
COPY --from=builder /out/migrate /app/migrate
COPY --from=builder /out/seeder /app/seeder
COPY --from=builder /app/docs ./docs
COPY --from=builder /app/internal/migration ./migrations

EXPOSE 8081

CMD ["./server"]