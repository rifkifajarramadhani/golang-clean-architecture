FROM golang:1.26.4-alpine AS build

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN mkdir -p /out && \
    CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/server ./cmd/server && \
    CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/worker ./cmd/worker && \
    CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/scheduler ./cmd/scheduler && \
    CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/queue ./cmd/queue && \
    CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/schedule ./cmd/schedule

FROM build AS development
RUN go install github.com/air-verse/air@v1.65.3 && \
    go install -tags mysql github.com/golang-migrate/migrate/v4/cmd/migrate@v4.19.1
WORKDIR /app
COPY . .
CMD ["air", "-c", ".air.toml"]

FROM alpine:3.22 AS production
RUN apk add --no-cache ca-certificates tzdata && \
    addgroup -S app && adduser -S -G app app && \
    mkdir -p /app && chown app:app /app
WORKDIR /app
COPY --from=build /out/ /usr/local/bin/
COPY --chown=app:app configs ./configs
USER app
EXPOSE 8080
HEALTHCHECK --interval=30s --timeout=3s --start-period=10s --retries=3 \
    CMD wget -q -O /dev/null http://127.0.0.1:8080/health/live || exit 1
CMD ["server"]
