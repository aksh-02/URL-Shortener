FROM golang:alpine AS builder

LABEL maintainer="<aksh0299@gmail.com>"

WORKDIR /app

COPY . .

RUN go build -o shortener .

FROM alpine

WORKDIR /urlshort

COPY --from=builder /app/ /urlshort/

ENV SHORTENED_URI mongodb+srv://urluser:urlpswd@cluster0.hl2el.mongodb.net/URLService?retryWrites=true&w=majority

CMD ["./shortener"]