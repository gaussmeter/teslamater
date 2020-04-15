FROM golang:1.13 AS builder
ENV GOPATH=
COPY main.go default.json go.mod ./
RUN curl -L -o pkger.tar.gz https://github.com/markbates/pkger/releases/download/v0.15.1/pkger_0.15.1_Linux_x86_64.tar.gz && \
    tar -zxvf *.tar.gz && \
    chmod u+x pkger && \
    ./pkger
#RUN go build -o main *.go
RUN CGO_ENABLED=0 GOOS=linux GOARCH=arm GOARM=6 go build -a -installsuffix cgo -o main *.go
FROM alpine:latest as ssl
RUN apk update && apk add ca-certificates
FROM scratch AS main
COPY --from=builder /go/main ./main
COPY --from=ssl /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
CMD ["./main"]
