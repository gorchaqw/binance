FROM golang:1.17-alpine3.14 as builder

RUN apk add --no-cache \
    make \
    build-base

RUN mkdir /project
ADD ./ /project/
WORKDIR /project
RUN go build -o binance ./cmd/


FROM alpine:3.14

RUN apk add --no-cache tzdata
ENV TZ=Europe/Moscow

COPY --from=builder /project/binance .
COPY --from=builder /project/.env .

CMD [ "./binance" ]