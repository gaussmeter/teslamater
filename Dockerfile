FROM golang:1.14.2 AS builder
COPY main.go default.json go.mod /go/src/teslamater/
WORKDIR /go/src/teslamater/
RUN curl -L -o pkger.tar.gz https://github.com/markbates/pkger/releases/download/v0.15.1/pkger_0.15.1_Linux_x86_64.tar.gz && \
    tar -zxvf *.tar.gz pkger && \
    ./pkger 
#RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -v -a -o main *.go 
RUN CGO_ENABLED=0 GOOS=linux GOARCH=arm GOARM=6 go build -v -a -installsuffix cgo -o main *.go
FROM alpine:latest as ssl
RUN apk update && apk add ca-certificates
FROM scratch AS main
COPY --from=builder /go/src/teslamater/main ./main
COPY --from=ssl /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
CMD ["./main"]
