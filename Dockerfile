FROM golang:1.14-buster as builder
WORKDIR /app

COPY ./ /app
RUN make build && ls -al

FROM debian
WORKDIR /opt/app/
#RUN apt-get update && apt-get install -y  curl net-tools
COPY --from=builder /app/bin /opt/app/
VOLUME ["/opt/app/config"]
CMD "/opt/app/block-explorer"
EXPOSE 8080
