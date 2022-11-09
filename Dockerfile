FROM golang:1.18.1-alpine as builder
ENV USER=appuser
ENV UID=10001
RUN apk update && apk add --no-cache git ca-certificates tzdata && update-ca-certificates
RUN mkdir -p /srv/bbb-convert /srv/bbb-convert/release
WORKDIR /srv/bbb-convert
COPY . .
RUN go get -d -v
RUN GOARCH=amd64 go build -o /srv/bbb-convert/release/bbb-convert

FROM alpine:3.11
RUN apk add --no-cache chromium ffmpeg
RUN adduser -D bigbluebutton bigbluebutton
COPY --from=builder /srv/bbb-convert/release /srv/bbb-convert
USER bigbluebutton
WORKDIR /home/bigbluebutton
ENTRYPOINT ["/srv/bbb-convert/bbb-convert"]