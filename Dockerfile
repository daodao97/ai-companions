FROM golang:1.24 AS builder

ARG BUILD_VERSION

WORKDIR /build

COPY . .
RUN GIT_TAG=$(git describe --tags --always) \
    && echo "GIT_TAG=${GIT_TAG}" \
    && CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags "-X main.Version=${GIT_TAG}" -o app .

FROM alpine:latest AS final

WORKDIR /app
COPY --from=builder /build/app /app/
COPY *.yaml /app/
COPY openapi.json /app/openapi.json
COPY assets /app/assets

RUN apk update && \
    apk add --no-cache sudo tzdata

ENV TZ=Asia/Shanghai
RUN ln -snf /usr/share/zoneinfo/$TZ /etc/localtime && echo $TZ > /etc/timezone

ENTRYPOINT ["/app/app"]