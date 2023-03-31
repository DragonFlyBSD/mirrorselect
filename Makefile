.if exists(.git/config)
VERSION!=git describe --tags --abbrev=0 HEAD
COMMIT!=git --no-pager log -1 --format=%h HEAD
DATE!=	git --no-pager log -1 --format=%cs HEAD

LDFLAGS=-X ${MODULE}/common.Version=${VERSION} \
	-X ${MODULE}/common.Commit=${COMMIT} \
	-X ${MODULE}/common.Date=${DATE}
.endif

.include "Makefile.inc"
