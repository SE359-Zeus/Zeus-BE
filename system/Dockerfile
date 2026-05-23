FROM golang:1.26.2-alpine AS build

WORKDIR /src

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

COPY --from=build /out/server /app/server
COPY --from=build /out/migrate /app/migrate
COPY --from=build /out/seeder /app/seeder
COPY docs ./docs
COPY migrations ./migrations

EXPOSE 8083

CMD ["./server"]