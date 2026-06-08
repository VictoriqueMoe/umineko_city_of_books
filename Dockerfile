ARG APP_VERSION=dev

FROM node:lts-alpine AS frontend-builder

ARG APP_VERSION
ENV VITE_APP_VERSION=$APP_VERSION

WORKDIR /app/frontend
COPY frontend/package.json frontend/package-lock.json ./
RUN npm ci
COPY frontend/ .
RUN npm run build && npm run build:ota

FROM golang:1.26-alpine AS builder

ARG APP_VERSION

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
COPY --from=frontend-builder /app/static/ ./static/

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-X umineko_city_of_books/internal/config.Version=${APP_VERSION}" -o main .

FROM alpine:latest

RUN apk add --no-cache ffmpeg libwebp-tools

WORKDIR /app

COPY --from=builder /app/main .

EXPOSE 4323

CMD ["./main"]
