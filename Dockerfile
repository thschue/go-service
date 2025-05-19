FROM golang:1.24

COPY . /src

WORKDIR /src

RUN go build .

CMD [ "go-service" ]
