@echo off
SET GOARCH=amd64
SET GOOS=linux
go build -o dlnaproxy
SET GOOS=windows
go build -o dlnaproxy.exe

SET GOARCH=arm
SET GOOS=linux
go build -o dlnaproxy_armhf

upx -9 dlnaproxy*