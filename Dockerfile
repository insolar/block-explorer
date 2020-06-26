FROM golang:1.14-buster as builder
WORKDIR /app

COPY ./ /app
RUN make vendor build

FROM debian
WORKDIR /opt/app/
COPY --from=builder /app/bin /opt/app/
RUN apt-get update && \
    apt-get install -y curl && \
    apt-get clean
CMD "/opt/app/block-explorer"
EXPOSE 8080
