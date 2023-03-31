MODULE=	github.com/DragonFlyBSD/mirrorselect

ifneq (,$(wildcard .git/config))
VERSION=$(shell git describe --tags --abbrev=0 HEAD)
COMMIT=	$(shell git --no-pager log -1 --format=%h HEAD)
DATE=	$(shell git --no-pager log -1 --format=%cs HEAD)

LDFLAGS=-X $(MODULE)/common.Version=$(VERSION) \
	-X $(MODULE)/common.Commit=$(COMMIT) \
	-X $(MODULE)/common.Date=$(DATE)
endif

all:
	CGO_ENABLED=0 go build -ldflags="$(LDFLAGS)" -o mirrorselect main.go

ci: all
	CGO_ENABLED=0 GOOS=dragonfly GOARCH=amd64 \
		    go build -ldflags="$(LDFLAGS)" -o mirrorselect main.go
	CGO_ENABLED=0 GOOS=freebsd GOARCH=amd64 \
		    go build -ldflags="$(LDFLAGS)" -o mirrorselect main.go

clean:
	rm -f mirrorselect

test: dbip
	go test -v ./common ./geoip ./monitor ./workerpool

dbip: testdata/dbip-city-lite.mmdb
testdata/dbip-city-lite.mmdb:
	curl https://download.db-ip.com/free/dbip-city-lite-2023-03.mmdb.gz | \
		gunzip > $@
