# Frontend: compiled here so one image serves both the SPA and the API from a
# single origin. REACT_APP_* are inlined at build time, so the API URL is
# relative -- same origin, no CORS.
FROM node:18-alpine AS web
WORKDIR /web
COPY frontend/package.json frontend/package-lock.json ./
RUN npm ci
COPY frontend/ ./
ARG REACT_APP_API_URL=/api
ARG REACT_APP_GOOGLE_CLIENT_ID=
ENV REACT_APP_API_URL=$REACT_APP_API_URL
ENV REACT_APP_GOOGLE_CLIENT_ID=$REACT_APP_GOOGLE_CLIENT_ID
ENV CI=true
RUN npm run build

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
COPY --from=web /web/build ./web

# Uploads are bind-mounted in compose; create it so the server can write even
# when no volume is attached.
RUN mkdir -p /app/uploads && chown -R app:app /app
USER app

EXPOSE 8080
CMD ["server"]
