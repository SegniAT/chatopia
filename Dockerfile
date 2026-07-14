# Stage 1: Build
FROM golang:1.26-alpine AS build-stage

WORKDIR /app

COPY go.mod go.sum ./

RUN go mod download

COPY . /app/

RUN go install github.com/a-h/templ/cmd/templ@v0.3.1020 && \
    templ generate

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o=/app/bin/linux_amd64/web ./cmd/web


# Stage 2: Deploy
FROM alpine:latest AS release-stage

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

