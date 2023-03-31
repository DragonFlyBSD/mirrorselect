ifneq (,$(wildcard .git/config))
VERSION=$(shell git describe --tags --abbrev=0 HEAD)
COMMIT=	$(shell git --no-pager log -1 --format=%h HEAD)
DATE=	$(shell git --no-pager log -1 --format=%cs HEAD)

LDFLAGS=-X $(MODULE)/common.Version=$(VERSION) \
	-X $(MODULE)/common.Commit=$(COMMIT) \
	-X $(MODULE)/common.Date=$(DATE)
endif

include Makefile.inc
