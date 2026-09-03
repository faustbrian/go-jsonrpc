SHELL := /usr/bin/env bash
GOLIB ?= golib

.PHONY: check ci cohesion interoperability inventory repository-check

check:
	$(GOLIB) check --all

ci:
	$(GOLIB) repository check
	$(GOLIB) cohesion check
	$(GOLIB) check --all

cohesion:
	$(GOLIB) cohesion check

interoperability:
	GOWORK=auto go -C interoperability test ./...

inventory repository-check:
	$(GOLIB) repository check
