FROM golang:1.18-alpine as builder

ADD . /speech_to_text_bot
WORKDIR /speech_to_text_bot

ENV GOPATH=/speech_to_text_bot
ENV GO111MODULE=auto
RUN go build -o /tmp/speech_to_text_bot ./src/speech_to_text_bot/cmd/speech_to_text_bot


FROM alpine:3.15

COPY --from=builder /tmp/speech_to_text_bot /usr/bin/speech_to_text_bot

RUN apk update
RUN apk add --no-cache ffmpeg

RUN echo '#! /bin/sh' > /usr/bin/entrypoint.sh
RUN echo 'speech_to_text_bot' >> /usr/bin/entrypoint.sh

RUN chmod +x /usr/bin/entrypoint.sh

ENTRYPOINT ["/usr/bin/entrypoint.sh"]
