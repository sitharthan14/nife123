FROM golang:1.13.4-alpine

RUN apk update && apk add --no-cache git

WORKDIR /app

RUN GO111MODULE=on CGO_ENABLED=0 GOOS=linux

COPY go.mod go.sum ./

RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -installsuffix cgo -o main .

EXPOSE 8080

CMD ["./main"]