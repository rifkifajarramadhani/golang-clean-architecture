FROM golang:1.26.4-alpine

WORKDIR /app

RUN go install github.com/air-verse/air@v1.65.3

RUN go install -tags 'mysql' github.com/golang-migrate/migrate/v4/cmd/migrate@v4.19.1

COPY go.mod go.sum ./

RUN go mod download

ENV PATH="$PATH:/go/bin"

COPY . .

CMD ["air", "-c", ".air.toml"]
