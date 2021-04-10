FROM golang:1.15-alpine AS build
# Support CGO and SSL
RUN apk --no-cache add gcc g++ make
RUN apk add git
WORKDIR /go/src/admes-backend
COPY . .
ENV GOPATH="/go/src/admes-backend"
RUN go get github.com/gorilla/mux
RUN go get github.com/gorilla/websocket
RUN GOOS=linux go build -ldflags="-s -w" -o main .

FROM alpine:3.10
RUN apk --no-cache add ca-certificates
WORKDIR /usr/bin
COPY --from=build /go/src/admes-backend/main .
EXPOSE 7777
ENTRYPOINT  ["./main"]
