FROM golang:1.14 AS builder
ENV GOPROXY https://goproxy.io
ENV CGO_ENABLED 0
WORKDIR /go/src/app
ADD . .
RUN go build -mod vendor -o /enforce-ns-annotations

FROM alpine:3.12
COPY --from=builder /enforce-ns-annotations /enforce-ns-annotations
CMD ["/enforce-ns-annotations"]