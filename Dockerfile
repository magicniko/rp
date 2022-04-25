FROM golang:alpine as build
RUN mkdir -p $GOPATH/src/github.com/magicniko/rp && apk update && apk add git
WORKDIR $GOPATH/src/github.com/magicniko/rp
ADD . .
RUN go build -o /rp

FROM alpine
COPY --from=build /rp /usr/local/bin/rp
RUN apk add --no-cache ca-certificates
