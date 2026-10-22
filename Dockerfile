# syntax=docker/dockerfile:1

FROM node:22-alpine AS web
WORKDIR /src/web
COPY web/package.json web/package-lock.json ./
RUN npm ci
COPY web/ ./
RUN npm run build

FROM golang:1.27.2-alpine AS go
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=web /src/web/dist ./web/dist
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/dockerity ./cmd/dockerity

# Dockerity runs as root on purpose: it needs the Docker socket, which is
# root-equivalent anyway (docs/design.md section 11).
FROM gcr.io/distroless/static-debian12
COPY --from=go /out/dockerity /usr/local/bin/dockerity
ENV DOCKERITY_LISTEN_ADDR=:8080 \
    DOCKERITY_DATA_DIR=/data
VOLUME ["/data", "/backups"]
EXPOSE 8080
HEALTHCHECK --interval=15s --timeout=5s --start-period=10s --retries=3 \
    CMD ["/usr/local/bin/dockerity", "healthcheck"]
ENTRYPOINT ["/usr/local/bin/dockerity"]
