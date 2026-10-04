#!/usr/bin/env bash
rm -rf ../patch*.so
#go build -gcflags "all=-N -l" -buildmode=plugin -o ../patch.so plugin.go
go build -buildmode=plugin -o ../patch.so plugin.go
