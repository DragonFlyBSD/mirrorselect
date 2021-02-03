DragonFly pkg mirrorselect
--------------------------

**Mirrorselect** is an HTTP backend service that selects the pkg(8) mirrors
according to their "distances" to the client.

The "distance" of a mirror is determined by:

* whether locate in the same country as the client
* whether locate on the same continent as the client
* great-circle distance between its coordinate and the client's

The selected mirrors and their ordering are:

* Prefer mirrors of the same **country** as the client.
* If not, then prefer mirrors of the same **continent**.
* If not, fallback to the *default* mirror.
* If multiple mirrors in the same country/continent, order them by
  *distance* to the client (calculated via latitude/longitude).
* Append the *default* mirror to the last as fallback.
* If cannot determine client's location, just return the *default* mirror.

Features
--------
* Simple and small:
  - simple config files
  - few direct dependencies:
  [gin-gonic/gin](https://github.com/gin-gonic/gin),
  [oschwald/maxminddb-golang](https://github.com/oschwald/maxminddb-golang),
  [spf13/viper](https://github.com/spf13/viper),
  [jlaffaye/ftp](https://github.com/jlaffaye/ftp)
* Stand-alone:
  - use offline IP geolocation database
  - [MaxMind DB format](https://maxmind.github.io/MaxMind-DB/)
  - support both [MaxMind](https://www.maxmind.com) and
  [DB-IP](https://db-ip.com) dataset
* Built-in mirror monitor:
  - periodically check mirror status
  - support HTTP, HTTPS and FTP
  - use a hysteresis to smooth status flipping

Implementation
--------------
This mirror selection implementation leverages pkg(8)'s **HTTP**
mirror type of repository.
Basically, the repository URL provides a sequence of lines beginning with
`URL:` followed by any amount of white space and one URL for a repository
mirror.
Then, pkg(8) tries these mirrors in the order listed until a download
succeeds.
See also: [pkg-repository(5)](https://man.dragonflybsd.org/?command=pkg-repository&section=5)

For example, configure the pkg repos as:
```
AUTO: {
	url: https://pkg.dragonflybsd.org/pkg/${ABI}/LATEST
	mirror_type: HTTP
}
```

And upon client's request, the service returns, e.g.,:
```
URL: https://mirror.sjtu.edu.cn/dragonflybsd/dports/dragonfly:5.10:x86:64/LATEST
URL: ...
URL: https://mirror-master.dragonflybsd.org/dports/dragonfly:5.10:x86:64/LATEST
```

Deployment
----------
1. Prepare the mirror list file `mirrors.toml`, listing all available
   pkg(8) mirrors and their locations.
2. Obtain one of the following **free** IP geolocation database
   (choose **MMDB** binary format):
   * [DB-IP Lite data](https://db-ip.com/db/download/ip-to-city-lite)
     <br>
     *Recommended*, more entries and higher precision, no sign-up required.
   * [MaxMind GeoLite2 data](https://dev.maxmind.com/geoip/geoip2/geolite2/)
     <br>
     NOTE: sign-up required to download the database.
3. Create the main config file `mirrorselect.toml`.
4. Run the **mirrorselect** service as a normal user (e.g., `nobody`).
5. Publish this service via Nginx/Apache.

License
-------
The 3-Clause BSD License

Copyright (c) 2021 The DragonFly Project.
