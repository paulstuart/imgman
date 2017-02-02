
SHELL := /bin/bash

PKG=$(PWD)/docker/dcman-pkg/shared

all: install

install:
	cp scripts/pxestats /usr/local/bin/
	cp init.d/pxestats /etc/init.d/pxestats
	echo 'PATH=$$PATH:/usr/local/bin' > /etc/sysconfig/pxestats
	chkconfig pxestats on
	service pxestats restart

server:
	go build

.PHONY: all install server

