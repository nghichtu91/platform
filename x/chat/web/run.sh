#!/bin/bash

if [ $1 == "build" ]
then
    go build
fi

if [ $1 == "race" ]
then
    go build -race
fi

if [ $? == 0 ]
then
    rm -rf RUNNING
    TZ=Asia/Shanghai ./web allinone
fi




