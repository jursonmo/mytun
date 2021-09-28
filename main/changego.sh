#!/bin/sh
sudo rm /usr/local/go
case $1 in
	12)
	ver="go1.12.10"
	;;
	13)
	ver="go1.13.4"
	;;
	14)
	ver="go1.14.12"
	;;
	15)
	ver="go1.15.5"
	;;
esac
sudo ln -s /usr/local/$ver /usr/local/go
echo `go version`
