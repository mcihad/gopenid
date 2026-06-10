FROM node:24-alpine AS frontend
WORKDIR /src/frontend
COPY frontend/package*.json ./
RUN npm ci
COPY frontend ./
RUN npm run build

FROM golang:1.26-alpine AS backend
WORKDIR /src
RUN apk add --no-cache git
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=frontend /src/internal/web/dist ./internal/web/dist
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/gopenid ./cmd/server

FROM alpine:3.23
WORKDIR /app
RUN adduser -D -H -u 10001 gopenid
COPY --from=backend /out/gopenid /app/gopenid
USER gopenid
EXPOSE 8080
ENTRYPOINT ["/app/gopenid"]
