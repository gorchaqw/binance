#!/bin/sh

file_name="./.version"
image_name="binance/binance"

build(){
  image_name=$1
  version=$2

  docker --context firstvds build -t "$image_name":"$version" .
}

deploy(){
  version=$1

  export BINANCE_APP_PORT="8080"
  export BINANCE_VERSION=$version
  export BINANCE_PG_INIT_DB="/root/db.sql"

  docker-compose --context firstvds up --remove-orphans -d
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

