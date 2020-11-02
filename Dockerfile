FROM golang:1.14 AS builder
ENV GOPROXY https://goproxy.io
ENV CGO_ENABLED 0
WORKDIR /go/src/app
ADD . .
RUN go build -o /enforce-qcloud-fixed-ip

FROM alpine:3.12
COPY --from=builder /enforce-qcloud-fixed-ip /enforce-qcloud-fixed-ip
CMD ["/enforce-qcloud-fixed-ip"]