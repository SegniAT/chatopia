# Stage 1: Build
FROM golang:1.23-alpine AS build-stage

WORKDIR /app

COPY go.mod go.sum ./

RUN go mod download

COPY . /app/

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o=/app/bin/linux_amd64/web ./cmd/web


# Stage 2: Deploy
FROM alpine:3.19 AS release-stage

RUN apk add --no-cache ca-certificates tzdata


WORKDIR /app

COPY --from=build-stage /app/bin/linux_amd64/web /app/
COPY --from=build-stage /app/ui /app/ui
COPY --from=build-stage /app/public /app/public

EXPOSE 4000

ENV APP_PORT=4000
ENV REDIS_ADDR=redis:6379
ENV ENV=development

CMD ["/app/web"]

