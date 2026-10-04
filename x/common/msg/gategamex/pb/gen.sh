#!/bin/sh
../../../../../planx/bin/protoc --plugin=../../../../../planx/bin/protoc-gen-go --proto_path=. --go_out=plugins=grpc,paths=source_relative:. proto/*.proto
mv -f proto/*.pb.go .

#protoc -I helloworld/ helloworld/helloworld.proto --go_out=plugins=grpc:helloworld

