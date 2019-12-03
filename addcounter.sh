#!/usr/bin/env bash
version=`cat buildcounter.txt`
version=$((version+1))
echo "$version" > buildcounter.txt
