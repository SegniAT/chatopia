# Stage 1: Build
FROM golang:1.22-alpine AS build-stage

WORKDIR /app

COPY go.mod go.sum ./

RUN go mod download

COPY . /app/

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o=/app/bin/linux_amd64/web ./cmd/web


# Stage 2: Deploy
FROM golang:1.22-alpine AS release-stage


WORKDIR /app

COPY --from=build-stage /app/bin/linux_amd64/web /app/
COPY --from=build-stage /app/ui /app/ui
COPY --from=build-stage /app/public /app/public

EXPOSE 5000

CMD [ "/app/web", "-port=5000"]

