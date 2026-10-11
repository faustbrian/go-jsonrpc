SHELL := /usr/bin/env bash
GOLIB ?= golib

.PHONY: check check-full ci cohesion interoperability inventory repository-check

check:
	$(GOLIB) check --local --all

check-full:
	$(GOLIB) check --all

ci:
	$(GOLIB) repository check
	$(GOLIB) cohesion check
	$(GOLIB) check --local --all

cohesion:
	$(GOLIB) cohesion check

interoperability:
	GOWORK=auto go -C interoperability test ./...

inventory repository-check:
	$(GOLIB) repository check
