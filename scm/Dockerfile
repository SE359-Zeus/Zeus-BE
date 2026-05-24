FROM golang:1.25-alpine AS builder

WORKDIR /src

RUN apk add --no-cache build-base

# Copy entire repository into the image. Build with context set to the repo root:
# docker build -f scm/Dockerfile -t zeus-scm-service .
COPY . .

WORKDIR /src/scm
RUN go mod download

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
COPY --from=builder /src/scm/docs ./docs
COPY --from=builder /src/scm/migration ./migrations

EXPOSE 8081

CMD ["./server"]