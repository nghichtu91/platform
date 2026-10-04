#!/usr/bin/env bash

GOOS=linux GOARCH=amd64 go build
#go build

rm -f e3w.zip

mkdir bin
mv e3w ./bin
mkdir ./bin/static
mkdir ./bin/static/dist
cp ./static/dist/* ./bin/static/dist/
mkdir ./bin/conf
cp ./conf/config.default.ini ./bin/conf/

zip -r e3w.zip ./bin

rm -rf bin