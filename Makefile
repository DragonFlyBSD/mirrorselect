all:
	CGO_ENABLED=0 go build -o mirrorselect main.go

ci: all
	CGO_ENABLED=0 GOOS=dragonfly GOARCH=amd64 \
		    go build -o mirrorselect main.go
	CGO_ENABLED=0 GOOS=freebsd GOARCH=amd64 \
		    go build -o mirrorselect main.go

clean:
	rm -f mirrorselect

test: dbip
	go test -v ./common ./geoip ./monitor ./workerpool

dbip: testdata/dbip-city-lite.mmdb
testdata/dbip-city-lite.mmdb:
	curl https://download.db-ip.com/free/dbip-city-lite-2023-03.mmdb.gz | \
		gunzip > $@
