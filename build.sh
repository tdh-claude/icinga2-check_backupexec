#!/usr/bin/env bash
version=`cat buildcounter.txt`
version=$((version+1))
echo "$version" > buildcounter.txt
go build -ldflags "-X main.buildcount=`echo $version`" -o check_backupexec