#!/bin/sh

file_name="./.version"
image_name="binance/binance"

build(){
  image_name=$1
  version=$2

  docker --context beget build -t "$image_name":"$version" .
}

deploy(){
  version=$1

  scp db.sql root@62.113.104.230:/root/db.sql
  scp loki-config.yaml root@62.113.104.230:/root/loki-config.yaml
  scp prometheus-config.yaml root@62.113.104.230:/root/prometheus-config.yaml

  export BINANCE_APP_PORT="8080"
  export BINANCE_APP_NAME="Binance_PROD"
  export BINANCE_VERSION=$version
  export BINANCE_PG_INIT_DB="/root/db.sql"
  export BINANCE_LOKI="/root/loki-config.yaml"
  export BINANCE_PROMETHEUS="/root/prometheus-config.yaml"
  export BINANCE_LOG_LEVEL="ERROR"

  docker-compose --context beget up --remove-orphans -d
}

gitRepo(){
  version=$1

  git commit -am"update to v.${version}"
  git push origin main
}


current_version=$(cat ${file_name})
echo "current version:  $current_version"

if [ "$1" == "current" ]; then
  deploy "$current_version"
  exit
fi

IFS='.' read -r -a version <<< "$current_version"

majorVersion="${version[0]}"
minorVersion="${version[1]}"
patchVersion="${version[2]}"

case $1 in
  "major")
    majorVersion=$(($majorVersion+1))
    minorVersion=0
    patchVersion=0
    ;;

  "minor")
    minorVersion=$(($minorVersion + 1))
    patchVersion=0
    ;;

  "patch")
    patchVersion=$(($patchVersion + 1))
    ;;

  *)
    echo "wrong argument..."
    exit
    ;;
esac

new_version="${majorVersion}.${minorVersion}.${patchVersion}"

echo "updated version:  $new_version"

build "$image_name" "$new_version"
deploy "$new_version"
gitRepo "$new_version"


echo "$new_version" > $file_name

