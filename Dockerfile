FROM golang:1.24-alpine3.21 AS builder
ENV USER=appuser
ENV UID=10001
RUN apk update && apk add --no-cache git ca-certificates tzdata && update-ca-certificates
RUN mkdir -p /srv/bbb-convert /srv/bbb-convert/release
WORKDIR /srv/bbb-convert
COPY . .
RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags='-w -s -extldflags "-static"' -a \
        -o /srv/bbb-convert/release/bbb-convert

FROM alpine:3.21
RUN apk add --no-cache chromium ffmpeg
RUN adduser -D bigbluebutton bigbluebutton
COPY --from=builder /srv/bbb-convert/release /srv/bbb-convert
USER bigbluebutton
WORKDIR /home/bigbluebutton
ENTRYPOINT ["/srv/bbb-convert/bbb-convert"]