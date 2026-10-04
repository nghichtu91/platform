#!/bin/sh
clientPath=${Jws2ProtoClientPath}
protoc_gen_go_dir=../../bin/protoc-gen-go

../../bin/protoc --plugin=${protoc_gen_go_dir} -I proto/  --go_out=. --go_opt=paths=source_relative proto/*.proto

../../bin/protoc --csharp_out=${clientPath}/server_battle_check/battlecheck/protogen proto/*.proto
