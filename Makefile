SHELL := /usr/bin/env bash
GOLIB ?= golib

.PHONY: check ci interoperability inventory repository-check

check:
	$(GOLIB) check --all

ci:
	$(GOLIB) repository check
	$(GOLIB) check --all

interoperability:
	GOWORK=auto go -C interoperability test ./...

inventory repository-check:
	$(GOLIB) repository check
