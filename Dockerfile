FROM golang:1.26-alpine

WORKDIR /app

RUN go install github.com/air-verse/air@latest

RUN go install -tags 'mysql' github.com/golang-migrate/migrate/v4/cmd/migrate@latest

COPY go.mod go.sum ./

RUN go mod download

ENV PATH="$PATH:/go/bin"

COPY . .

CMD ["air", "-c", ".air.toml"]