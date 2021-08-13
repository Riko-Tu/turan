FROM golang:latest as builder

WORKDIR /go/src/TEFS-BE/
COPY . ./
ENV GOPROXY=https://goproxy.cn GO111MODULE=on
RUN go mod vendor
RUN CGO_ENABLED=0 GOOS=linux go build -mod=vendor -o server pkg/latex/service/service.go

FROM ccr.ccs.tencentyun.com/tefs/ubuntu-tectonic:latest

RUN mkdir /app
WORKDIR /app
COPY --from=builder /go/src/TEFS-BE/server .

CMD ["./server"]