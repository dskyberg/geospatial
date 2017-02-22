#!/usr/bin/env bash
CONTAINER_NAME='mysql_geo'
DATA_DIR="~/mysql_data"
PASSWORD='password'
TAG='5.7'

docker run --name $CONTAINER_NAME -v $DATA_DIR:/var/lib/mysql -p 3306:3306 -e MYSQL_ROOT_PASSWORD=$PASSWORD -d mysql/mysql-server:$TAG
