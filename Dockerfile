FROM golang:1.26.4-alpine3.23 AS build

ARG TARGET=server
WORKDIR /src
RUN apk add --no-cache ca-certificates tzdata
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -buildvcs=false -trimpath -ldflags="-s -w" -o /out/service ./cmd/${TARGET}

FROM alpine:3.23
RUN apk add --no-cache ca-certificates tzdata && addgroup -S app && adduser -S -G app app
COPY --from=build /out/service /service
USER app
EXPOSE 8080
ENTRYPOINT ["/service"]
