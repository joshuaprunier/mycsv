#!/bin/bash

major_rev="1"
minor_rev="0"
commit_num=`git shortlog -s | awk '{ sum += $1 } END { print sum }'`
gitsha=`git log -n1 --pretty="%h"`
date=`date +"%Y%m%d"`
time=`date +"%k%M%S"`

version=${major_rev}.${minor_rev}.${commit_num}-${gitsha}-${date}.${time}

echo
echo "Building mycsv version"
echo ${version}
echo


# Linux
echo
echo "Building Linux"
mkdir -p bin/linux
GOOS=linux GOARCH=amd64 go build -ldflags "-X main.versionInformation=$version" -o bin/linux/mycsv mycsv.go csv_writer.go reset_unix.go
if [[ $? -eq 0 ]]; then
	echo "	mycsv - OK"
else
	echo "	mycsv - FAILED"
fi

# Windows
echo
echo "Building Windows"
mkdir -p bin/windows
GOOS=windows GOARCH=amd64 go build -ldflags "-X main.versionInformation=$version" -o bin/windows/mycsv.exe mycsv.go csv_writer.go reset_win.go
if [[ $? -eq 0 ]]; then
	echo "	mycsv.exe - OK"
else
	echo "	mycsv.exe - FAILED"
fi

# Darwin
echo
echo "Building Darwin"
mkdir -p bin/darwin
GOOS=darwin GOARCH=amd64 go build -ldflags "-X main.versionInformation=$version" -o bin/darwin/mycsv mycsv.go csv_writer.go reset_unix.go
if [[ $? -eq 0 ]]; then
	echo "	mycsv - OK"
else
	echo "	mycsv - FAILED"
fi

echo
echo "Done!"
