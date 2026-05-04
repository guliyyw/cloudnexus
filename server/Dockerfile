FROM golang:alpine AS builder

ARG SERVICE=user-file-svc

ENV GOPROXY=https://goproxy.cn,direct

RUN apk add --no-cache git

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /bin/service ./cmd/${SERVICE}

FROM alpine:latest

RUN apk add --no-cache ca-certificates tzdata curl

COPY --from=builder /bin/service /usr/local/bin/service

CMD ["service"]
