FROM golang:alpine AS builder

LABEL maintainer="<aksh0299@gmail.com>"

WORKDIR /app

COPY . .

RUN go build -o shortener .

FROM alpine

WORKDIR /urlshort

COPY --from=builder /app/ /urlshort/

ENV SHORTENED_URI 

CMD ["./shortener"]