FROM golang:latest AS builder
RUN go get github.com/eclipse/paho.mqtt.golang 
RUN go get github.com/gaussmeter/mqttagent
RUN go get github.com/sirupsen/logrus
RUN go get github.com/thanhpk/randstr
COPY main.go main.go 
RUN CGO_ENABLED=0 GOOS=linux GOARCH=arm GOARM=6 go build -a -installsuffix cgo -o main .
FROM alpine:latest as ssl
RUN apk update && apk add ca-certificates
FROM scratch AS main
COPY --from=builder /go/main ./main
COPY --from=ssl /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY config.json ./config.json
CMD ["./main"]
