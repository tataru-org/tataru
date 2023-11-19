FROM golang:1.19.5-alpine3.17

WORKDIR /app

COPY ./go.mod ./
COPY ./go.sum ./
RUN go mod download && \
    mkdir /app/initial-db-data

COPY ./*.go ./

RUN go build -o /app/tataru

ENTRYPOINT [ "/app/tataru" ]
