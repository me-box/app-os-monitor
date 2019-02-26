FROM golang:1.11.5-alpine3.8 as gobuild
WORKDIR /
ENV GOPATH="/"
RUN apk update && apk add pkgconfig build-base bash autoconf automake libtool gettext openrc git libzmq zeromq-dev mercurial
#COPY . . if you update the libs below build with --no-cache
RUN go get -u github.com/gorilla/mux
RUN go get -u gonum.org/v1/gonum/...
RUN go get -u gonum.org/v1/plot/...
RUN go get -u github.com/me-box/lib-go-databox
COPY . .
RUN addgroup -S databox && adduser -S -g databox databox
RUN GGO_ENABLED=0 GOOS=linux go build -a -tags netgo -installsuffix netgo -ldflags '-s -w' -o app /src/*.go

FROM alpine:3.8
COPY --from=gobuild /etc/passwd /etc/passwd
RUN apk update && apk add libzmq
USER databox
WORKDIR /
COPY --from=gobuild /app .
LABEL databox.type="app"
EXPOSE 8080

CMD ["./app"]
#CMD ["sleep","2147483647"]
