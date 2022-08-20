# RUN
# docker build -t binance/binance:0.0.2 .

# docker-compose -H "ssh://root@37.46.130.3"  up --force-recreate --build
# docker-compose up --force-recreate --build

FROM golang:1.17-alpine3.14 as builder

RUN apk add --no-cache \
    make \
    build-base

RUN mkdir /project
ADD ./ /project/
WORKDIR /project
RUN go build -o binance ./cmd/


FROM alpine:3.14
COPY --from=builder /project/binance .
COPY --from=builder /project/.env .

CMD [ "./binance" ]