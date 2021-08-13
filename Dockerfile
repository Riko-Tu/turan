FROM golang:latest as builder

WORKDIR /go/src/TEFS-BE/
COPY . ./
ENV GOPROXY=https://goproxy.cn GO111MODULE=on
RUN go mod vendor
RUN CGO_ENABLED=0 GOOS=linux go build -mod=vendor -o server cmd/admin/service/service.go

FROM alpine:3.12

RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.ustc.edu.cn/g' /etc/apk/repositories
RUN apk update
RUN apk add curl bash tree tzdata
RUN cp -r -f /usr/share/zoneinfo/Hongkong /etc/localtime
RUN echo -ne "Alpine Linux 3.4 image. (`uname -rsv`)\n" >> /root/.built
RUN apk add ca-certificates
RUN mkdir /app
WORKDIR /app
COPY --from=builder /go/src/TEFS-BE/server .

CMD ["./server"]