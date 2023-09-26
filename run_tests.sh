#!/bin/bash

go mod tidy
go test -v -count=1 ./...
