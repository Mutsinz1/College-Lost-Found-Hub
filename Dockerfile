# Backend: build the server and migrate binaries, then ship them on a small
# runtime image. The migrations and seed SQL are copied in because the migrate
# binary reads them from disk at runtime.
FROM golang:1.25-alpine AS build
WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY cmd ./cmd
COPY internal ./internal
RUN CGO_ENABLED=0 go build -o /out/server ./cmd/server \
 && CGO_ENABLED=0 go build -o /out/migrate ./cmd/migrate \
 && CGO_ENABLED=0 go build -o /out/admin ./cmd/admin

FROM alpine:3.19
RUN apk add --no-cache ca-certificates \
 && adduser -D -u 10001 app
WORKDIR /app

COPY --from=build /out/server /out/migrate /out/admin /usr/local/bin/
COPY migrations ./migrations

# Uploads are bind-mounted in compose; create it so the server can write even
# when no volume is attached.
RUN mkdir -p /app/uploads && chown -R app:app /app
USER app

EXPOSE 8080
CMD ["server"]
