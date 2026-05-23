FROM golang:1.26.2-alpine AS builder

WORKDIR /app

RUN apk add --no-cache ca-certificates git

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/server ./cmd/server && \
	CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/migrate ./cmd/migrate && \
	CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/seeder ./cmd/seeder

FROM alpine:3.22

RUN apk add --no-cache ca-certificates

WORKDIR /app

ARG VERSION=dev

LABEL org.opencontainers.image.version=$VERSION

COPY --from=builder /out/server ./server
COPY --from=builder /out/migrate ./migrate
COPY --from=builder /out/seeder ./seeder
COPY --from=builder /app/docs ./docs

EXPOSE 8082

CMD ["./server"]
