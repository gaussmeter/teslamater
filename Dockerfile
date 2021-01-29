FROM golang:1.16-rc AS builder
COPY main.go default.json go.mod go.sum /go/src/teslamater/
WORKDIR /go/src/teslamater/
#RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -v -a -o main *.go 
RUN CGO_ENABLED=0 GOOS=linux GOARCH=arm GOARM=6 go build -v -a -installsuffix cgo -o main *.go
FROM alpine:latest as ssl
RUN apk update && apk add ca-certificates
FROM scratch AS main
COPY --from=builder /go/src/teslamater/main ./main
COPY --from=ssl /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
CMD ["./main"]
