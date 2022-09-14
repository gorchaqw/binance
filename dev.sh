#!/bin/sh


# 10826 grafana prometheus go metrics

deploy_local(){
  export BINANCE_VERSION="dev"
  export BINANCE_PG_INIT_DB="./db.sql"
  export BINANCE_LOKI="./loki-config.yaml"
  export BINANCE_PROMETHEUS="./prometheus-config.yaml"
  export BINANCE_APP_PORT="8080"
  export BINANCE_APP_NAME="Binance_DEV"
  export BINANCE_LOG_LEVEL="DEBUG"

  docker build --no-cache -t binance/binance:dev .
  docker-compose up --remove-orphans -d
}

deploy_dev(){
  scp db.sql root@62.113.99.249:/root/db.sql
  scp loki-config.yaml root@62.113.99.249:/root/loki-config.yaml
  scp prometheus-config.yaml root@62.113.99.249:/root/prometheus-config.yaml

  export BINANCE_APP_PORT="8080"
  export BINANCE_APP_NAME="Binance_Beget_DEV"
  export BINANCE_VERSION="dev"
  export BINANCE_PG_INIT_DB="/root/db.sql"
  export BINANCE_LOKI="/root/loki-config.yaml"
  export BINANCE_PROMETHEUS="/root/prometheus-config.yaml"
  export BINANCE_LOG_LEVEL="ERROR"

  docker --context beget build --no-cache -t binance/binance:dev .
  docker-compose --context beget up --remove-orphans -d
}

case $1 in
  "dev")
   deploy_dev
    ;;

  "local")
   deploy_local
    ;;

  *)
    echo "wrong argument..."
    exit
    ;;
esac