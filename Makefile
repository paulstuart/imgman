
SHELL := /bin/bash

PKG=$(PWD)/docker/dcman-pkg/shared

all: install

install:
	cp scripts/pxestats /usr/local/bin/
	cp init.d/etc/init.d
	chkconfig pxestats on
	service pxestats restart

server:
	go build

.PHONY: all install server

