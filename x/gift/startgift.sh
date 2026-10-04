#!/bin/bash

run(){
    rm -rf RUNNING
    GORACE="halt_on_error=1" TZ=Asia/Shanghai ./gift allinone
}
buildSuccessThenRun(){
    if [[ $? == 0 ]]
    then
        echo "build success!"
        run
    fi
}

if [[ $1 == "build" ]]
then
    go build
    buildSuccessThenRun
fi

if [[ $1 == "race" ]]
then
    go build -race
    buildSuccessThenRun
fi

if [ $1 == "run" ]
then
    run
fi

buildSuccessThenRun(){
    if [ $? == 0 ]
    then
        echo "build success!"
        run
    fi
}