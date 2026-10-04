#!/usr/bin/env bash

protoc --descriptor_set_out=./packet.protodesc --proto_path=./ ./proto/packet.proto
mono `which protogen.exe` -i:./packet.protodesc -o:./packet.cs