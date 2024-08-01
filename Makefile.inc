PROG=		mirrorselect
MODULE=		github.com/DragonFlyBSD/$(PROG)

BUILD_ARGS+=	-trimpath -ldflags "$(LDFLAGS)"

all:
	env CGO_ENABLED=0 \
		go build $(BUILD_ARGS) -o $(PROG)

ci: all
	env CGO_ENABLED=0 GOOS=dragonfly GOARCH=amd64 \
		go build $(BUILD_ARGS) -o $(PROG)-dragonfly-amd64
	env CGO_ENABLED=0 GOOS=freebsd GOARCH=amd64 \
		go build $(BUILD_ARGS) -o $(PROG)-freebsd-amd64

clean:
	rm -f $(PROG) $(PROG)-*-*

test: dbip
	go test -v ./common ./geoip ./monitor ./workerpool

dbip: testdata/dbip-city-lite.mmdb
testdata/dbip-city-lite.mmdb:
	curl https://download.db-ip.com/free/dbip-city-lite-2023-03.mmdb.gz | \
		gunzip > $@
