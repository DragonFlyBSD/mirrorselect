DragonFly pkg mirrorselect
--------------------------

**Mirrorselect** is an HTTP backend service that selects the pkg(8) mirrors
according to their "distances" to the client.

The "distance" of a mirror is determined by:
- whether locate in the same country as the client
- whether locate on the same continent as the client
- great-circle distance between its coordinate and the client's

The selected mirrors and their ordering are:
- Prefer mirrors of the same **country** as the client.
- If not, then prefer mirrors of the same **continent**.
- If not, fallback to the *default* mirror.
- If multiple mirrors in the same country/continent, order them by
  *distance* to the client (calculated via latitude/longitude).
- Append the *default* mirror to the last as fallback.
- If cannot determine client's location, just return the *default* mirror.

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
	url: https://pkg.dragonflybsd.org/
	mirror_type: HTTP
}
```

And upon client's request, the service returns, e.g.,:
```
URL: https://mirror.sjtu.edu.cn/dragonflybsd/dports/${ABI}/LATEST
URL: ...
URL: https://mirror-master.dragonflybsd.org/dports/${ABI}/LATEST
```

License
-------
The 3-Clause BSD License

Copyright (c) 2021 The DragonFly Project.

This product includes
[GeoLite2 data](https://dev.maxmind.com/geoip/geoip2/geolite2/)
created by MaxMind, available from
[https://www.maxmind.com](https://www.maxmind.com).
