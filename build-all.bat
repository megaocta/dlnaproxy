@echo off
SET GOARCH=amd64
SET GOOS=linux
go build -o dlna_proxy
SET GOOS=windows
go build -o dlna_proxy.exe

SET GOARCH=arm
SET GOOS=linux
go build -o dlna_proxy_armhf

upx -9 dlna_proxy*