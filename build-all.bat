@echo off
SET GOARCH=amd64
SET GOOS=linux
ECHO Compile Linux
go build -o dlnaproxy
SET GOOS=windows
ECHO Compile Windows
go build -o dlnaproxy.exe

SET GOARCH=arm
SET GOOS=linux
ECHO Compile Linux ARM
go build -o dlnaproxy_armhf

start upx -9 dlnaproxy.exe
start upx -9 dlnaproxy
start upx -9 dlnaproxy_armhf