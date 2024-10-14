FROM golang:1.23-alpine

RUN apk add --no-cache git

WORKDIR /usr/src/app

# We want to populate the module cache based on the go.{mod,sum} files.
COPY go.mod .
COPY go.sum .

RUN go mod download

COPY . .

# Build the Go app
RUN go build -o ./bin/notion-site main.go

ENTRYPOINT ["/usr/src/app/bin/notion-site"]
