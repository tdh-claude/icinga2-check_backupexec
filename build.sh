#!/usr/bin/env bash
version=`cat buildcounter.txt`
version=$((version+1))
echo "$version" > buildcounter.txt
GOOS=windows GOARCH=amd64 go build -ldflags "-X main.buildcount=%VERSION%" -o check_backupexec.exe
GOOS=linux GOARCH=amd64 go build -ldflags "-X main.buildcount=`echo $version`" -o check_backupexec
