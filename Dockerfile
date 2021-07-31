FROM golang:1.16-alpine

RUN apk add tzdata && \
    cp /usr/share/zoneinfo/America/Vancouver /etc/localtime && \
    echo "America/Vancouver" > /etc/timezone && \
    date

WORKDIR /app

COPY go.mod ./
COPY go.sum ./
RUN go mod download

COPY . .

RUN go build -o /notionify cmd/main.go

EXPOSE 8000

CMD [ "/notionify" ]