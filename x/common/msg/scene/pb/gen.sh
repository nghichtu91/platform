#!/bin/sh
#protoc -I proto/ proto/*.proto --go_out=plugins=grpc:.
../../../../../planx/bin/protoc --plugin=../../../../../planx/bin/protoc-gen-go --proto_path=. --go_out=plugins=grpc,paths=source_relative:. proto/*.proto
mv -f proto/*.pb.go .
#../../../../../planx/bin/protoc --plugin=../../../../../planx/bin/protoc-gen-go  proto/g_s.proto --go_out=plugins=grpc:.


#protoc -I helloworld/ helloworld/helloworld.proto --go_out=plugins=grpc:helloworld