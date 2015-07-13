#!/bin/bash

# Linux
echo
echo "Building Linux"
mkdir -p bin/linux
go build -o bin/linux/mycsv mycsv.go csv_writer.go reset_unix.go
if [[ $? -eq 0 ]]; then
	echo "	mycsv - OK"
else
	echo "	mycsv - FAILED"
fi

# Windows
echo
echo "Building Windows"
mkdir -p bin/windows
GOOS=windows GOARCH=amd64 go build -o bin/windows/mycsv.exe mycsv.go csv_writer.go reset_win.go
if [[ $? -eq 0 ]]; then
	echo "	mycsv.exe - OK"
else
	echo "	mycsv.exe - FAILED"
fi

# Darwin
echo
echo "Building Darwin"
mkdir -p bin/darwin
GOOS=darwin GOARCH=amd64 go build -o bin/darwin/mycsv mycsv.go csv_writer.go reset_unix.go
if [[ $? -eq 0 ]]; then
	echo "	mycsv_mac - OK"
else
	echo "	mycsv_mac - FAILED"
fi

echo
echo "Done!"
