FROM golang:1.24 AS production

COPY . /src

WORKDIR /src

RUN go build .

CMD [ "go-service" ]
