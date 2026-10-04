#!/bin/bash

run(){
    GORACE="halt_on_error=1" GODEBUG=x509sha1=1 TZ=Asia/Shanghai ./gatex gate -c config2.toml
}

buildSuccessThenRun(){
    if [ $? == 0 ]
    then
        echo "build success!"
        run
    fi
}

if [ $1 == "build" ]
then
    go build
    buildSuccessThenRun
fi

if [ $1 == "race" ]
then
    go build -race
    buildSuccessThenRun
fi

if [ $1 == "run" ]
then
    run
fi
