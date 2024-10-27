FROM node:20-alpine AS BuildFrontend
WORKDIR /app
COPY ./internal/web/static/ /app/internal/web/static/
WORKDIR /app/internal/web/static
RUN npm ci
RUN npm run build

FROM golang:1.21-bookworm AS BuildBackend
RUN apt-get update \
    && apt-get install --no-install-recommends --assume-yes \
    build-essential
WORKDIR /app
COPY ./cmd/logsuck/main.go ./cmd/logsuck/main.go
COPY ./internal/ ./internal/
COPY ./pkg/ ./pkg/
COPY ./plugins/ ./plugins/
COPY ./go.mod ./
COPY ./go.sum ./
COPY --from=BuildFrontend /app/internal/web/static/dist/ /app/internal/web/static/dist/
RUN CGO_ENABLED=1 go build -ldflags '-s -w' -trimpath -o /dist/logsuck ./cmd/logsuck/main.go
RUN ldd /dist/logsuck | tr -s [:blank:] '\n' | grep ^/ | xargs -I % install -D % /dist/%
RUN useradd -u 1001 -m logsuck

FROM scratch
WORKDIR /
COPY --from=BuildBackend /dist /
COPY --from=BuildBackend /etc/passwd /etc/passwd
USER 1001
EXPOSE 8080
ENTRYPOINT [ "/logsuck" ]
