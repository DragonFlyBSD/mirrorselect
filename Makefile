all:
	GOOS=dragonfly GOARCH=amd64 go build -o mirrorselect main.go
	GOOS=freebsd GOARCH=amd64 go build -o mirrorselect main.go

test: dbip
	go test -v ./common ./geoip ./monitor ./workerpool

clean:
	rm -f mirrorselect

dbip: testdata/dbip-city-lite.mmdb
testdata/dbip-city-lite.mmdb:
	curl https://download.db-ip.com/free/dbip-city-lite-2023-03.mmdb.gz | \
		gunzip > $@
