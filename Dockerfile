FROM golang:1.14-buster as builder
WORKDIR /app

COPY ./ /app
RUN make build && ls -al

FROM debian
WORKDIR /opt/app/
COPY --from=builder /app/bin /opt/app/
VOLUME ["/opt/app/config"]
CMD "/opt/app/block-explorer"
EXPOSE 8080
