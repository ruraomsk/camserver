#!/bin/bash
echo "Start to Linux deploy"
go build
if [ $? -ne 0 ]; then
	echo 'An error has occurred! Aborting the script execution...'
	exit 1
fi
sudo cp camserver /home/rura/mnt/Irkutsk/camserver

