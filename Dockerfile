FROM golang:1.19-alpine as builer

ENV GOPROXY="https://goproxy.cn,direct"

WORKDIR /simple-web-server
COPY . ./
RUN CGO_ENABLED=0 go build -ldflags "-s -w"


FROM alpine

COPY --from=builer /simple-web-server/simple-web-server /simple-web-server

CMD [ "/simple-web-server" ]
