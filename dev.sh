#!/bin/sh


# ssh root@62.217.179.41
# docker context create beget --docker "host=ssh://root@62.217.179.41"

# 10826 grafana prometheus go metrics


debug(){
  export BINANCE_VERSION="dev"
   export BINANCE_PG_INIT_DB="./db.sql"
   export BINANCE_LOKI="./loki-config.yaml"
   export BINANCE_PROMETHEUS="./prometheus-config.yaml"
   export BINANCE_APP_PORT="8080"
   export BINANCE_APP_NAME="Binance_DEV"
   export BINANCE_LOG_LEVEL="DEBUG"

 docker-compose build --force-rm --no-cache
 docker-compose up --remove-orphans -d prometheus
}

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
  scp db.sql root@62.217.179.41:/root/db.sql

  export BINANCE_APP_PORT="8080"
  export BINANCE_APP_NAME="Binance_Beget_DEV"
  export BINANCE_VERSION="dev"
  export BINANCE_PG_INIT_DB="/root/db.sql"
  export BINANCE_LOKI="/root/loki-config.yaml"
  export BINANCE_PROMETHEUS="/root/prometheus-config.yaml"
  export BINANCE_LOG_LEVEL="DEBUG"

  docker context use beget

  docker build --no-cache -t binance/binance:dev .
  docker-compose up --remove-orphans -d

  docker context use default

#  docker --context beget build --no-cache -t binance/binance:dev .
#  docker-compose --context beget up --remove-orphans -d .
}

case $1 in
  "dev")
   deploy_dev
    ;;

 "d")
   debug
    ;;

  "local")
   deploy_local
    ;;

  *)
    echo "wrong argument..."
    exit
    ;;
esac