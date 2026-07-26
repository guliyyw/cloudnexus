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

ARG INSTALL_LIBREOFFICE=false
ARG INSTALL_FFMPEG=false

RUN apk add --no-cache ca-certificates tzdata curl && \
    if [ "$INSTALL_LIBREOFFICE" = "true" ]; then apk add --no-cache libreoffice; fi && \
    if [ "$INSTALL_FFMPEG" = "true" ]; then apk add --no-cache ffmpeg; fi && \
    addgroup -S cloudnexus && adduser -S cloudnexus -G cloudnexus

COPY --from=builder /bin/service /usr/local/bin/service

# Ensure log directory is writable by non-root user
RUN mkdir -p /app/logs /app/recordings && chown -R cloudnexus:cloudnexus /app/logs /app/recordings

USER cloudnexus

CMD ["service"]
