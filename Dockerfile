FROM golang:1.14-buster as builder
WORKDIR /app

COPY ./ /app
RUN make all && ls -al

FROM debian
WORKDIR /opt/app/
COPY --from=builder /app/bin /opt/app/
RUN apt-get update && \
    apt-get install -y curl \
    && rm -rf /var/cache/apk/*
VOLUME ["/opt/app/config"]
CMD "/opt/app/block-explorer"
EXPOSE 8080
