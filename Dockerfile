FROM gcr.io/distroless/static-debian10

ARG CMD_PATH

ENV ADDRESS "0.0.0.0:42700"

COPY ${CMD_PATH} /app

CMD ["/app"]