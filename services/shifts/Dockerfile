# syntax=docker/dockerfile:1
FROM golang:1.26-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /out/shifts ./cmd/server

FROM alpine:3.20
RUN apk add --no-cache ca-certificates && \
    adduser -D -u 10001 app
USER app
COPY --from=build /out/shifts /usr/local/bin/shifts
EXPOSE 8080
ENTRYPOINT ["/usr/local/bin/shifts"]
