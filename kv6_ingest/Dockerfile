FROM golang:1.26-alpine

WORKDIR /app

RUN apk add --no-cache gcc musl-dev zeromq-dev

COPY go.mod go.sum ./
RUN go mod download

COPY . . 
ENV CGO_ENABLED=1
RUN go build -v

CMD ["/app/kv6_ingest"]
