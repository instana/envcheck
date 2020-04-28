FROM golang:1.14-buster as build

ARG CMD_PATH
ARG GIT_SHA

WORKDIR /go/src/app
ADD . /go/src/app

RUN go get -d -v ./...
RUN go build -v -ldflags "-X main.Revision=$GIT_SHA" -o /go/bin/app $CMD_PATH

FROM gcr.io/distroless/base-debian10
COPY --from=build /go/bin/app /

ENV ADDRESS "0.0.0.0:42699"

CMD ["/app"]