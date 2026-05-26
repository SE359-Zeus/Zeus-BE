FROM golang:1.26.2-alpine AS build

WORKDIR /src/system

RUN apk add --no-cache build-base

COPY system/go.mod system/go.sum ./
COPY pkg/exception /src/pkg/exception
COPY pkg/response /src/pkg/response
RUN go mod download

COPY system/. .

ENV CGO_ENABLED=1

RUN go build -trimpath -ldflags="-s -w" -o /out/server ./cmd/server && \
	go build -trimpath -ldflags="-s -w" -o /out/migrate ./cmd/migrate && \
	go build -trimpath -ldflags="-s -w" -o /out/seeder ./cmd/seeder

FROM alpine:3.22

RUN apk add --no-cache ca-certificates sqlite-libs

RUN addgroup -S -g 1001 system && adduser -S -u 1001 -G system system

WORKDIR /app


ARG VERSION=dev

LABEL org.opencontainers.image.version=$VERSION

COPY --from=build /out/server /app/server
COPY --from=build /out/migrate /app/migrate
COPY --from=build /out/seeder /app/seeder
COPY --from=build /src/system/docs ./docs
COPY --from=build /src/system/migrations ./migrations
COPY --from=build /src/system/templates ./templates

RUN chown -R system:system /app

USER system

EXPOSE 8083

CMD ["./server"]