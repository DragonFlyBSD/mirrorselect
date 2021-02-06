all:
	go build -o mirrorselect main.go

test: dbip
	go test -v ./common ./geoip ./monitor ./workerpool

dbip: testdata/dbip-city-lite-2021-02.mmdb
testdata/dbip-city-lite-2021-02.mmdb:
	curl https://download.db-ip.com/free/dbip-city-lite-2021-02.mmdb.gz | \
		gunzip > $@
