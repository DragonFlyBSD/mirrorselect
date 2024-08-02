package monitor

import (
	"net/url"
	"testing"

	"github.com/DragonFlyBSD/mirrorselect/common"
)


func TestHttpCheck(t *testing.T) {
	var ok_urls = []string{
		"http://www.sjtu.edu.cn/",
		"https://www.sjtu.edu.cn/",
		"http://ftp.twaren.net/BSD/DragonFlyBSD/dports",
	}

	appConfig.Monitor.TLSVerify = true
	for _, utext := range ok_urls {
		u, _ := url.Parse(utext)
		status, err := httpCheck(u)
		if err != nil || !status {
			t.Errorf("httpCheck(%q) = (%v, %v); want %v\n",
					u, status, err, true)
		}
	}

	appConfig.Monitor.TLSVerify = false
	for _, utext := range ok_urls {
		u, _ := url.Parse(utext)
		status, err := httpCheck(u)
		if err != nil || !status {
			t.Errorf("httpCheck(%q) = (%v, %v); want %v\n",
					u, status, err, true)
		}
	}

	var fail_urls = []string{
		"http://xxx.not-exist.zzz/",
		"http://www.sjtu.edu.cn/xxx.not-exist.zzz/",
	}
	for _, utext := range fail_urls {
		u, _ := url.Parse(utext)
		status, err := httpCheck(u)
		if err == nil || status {
			t.Errorf("httpCheck(%q) = (%v, %v); want %v\n",
					u, status, err, false)
		}
	}

	var invalid_urls = []string{
		"ftp://ftp.sjtu.edu.cn/",
		"xxx://www.example.com/",
	}
	for _, utext := range invalid_urls {
		u, _ := url.Parse(utext)
		status, err := httpCheck(u)
		if err == nil || status {
			t.Errorf("httpCheck(%q) = (%v, %v); want %v\n",
					u, status, err, false)
		}
	}
}


func TestFtpCheck(t *testing.T) {
	var ok_urls = []string{
		"ftp://ftp.freebsd.org/",
		"ftp://ftp.freebsd.org/pub/FreeBSD/",
	}
	for _, utext := range ok_urls {
		u, _ := url.Parse(utext)
		status, err := ftpCheck(u)
		if err != nil || !status {
			t.Errorf("ftpCheck(%q) = (%v, %v); want %v\n",
					u, status, err, true)
		}
	}

	var fail_urls = []string{
		"ftp://xxx.not-exist.zzz/",
		"ftp://ftp.sjtu.edu.cn/xxx.not-exist.zzz/",
	}
	for _, utext := range fail_urls {
		u, _ := url.Parse(utext)
		status, err := ftpCheck(u)
		if err == nil || status {
			t.Errorf("ftpCheck(%q) = (%v, %v); want %v\n",
					u, status, err, false)
		}
	}

	var invalid_urls = []string{
		"http://ftp.sjtu.edu.cn/",
		"xxx://www.example.com/",
	}
	for _, utext := range invalid_urls {
		u, _ := url.Parse(utext)
		status, err := ftpCheck(u)
		if err == nil || status {
			t.Errorf("ftpCheck(%q) = (%v, %v); want %v\n",
					u, status, err, false)
		}
	}
}


func TestHysteresis(t *testing.T) {
	appConfig.Monitor.Hysteresis = 2
	mirror := &common.Mirror{
		Name: "Test",
		Status: common.MirrorStatus{
			Online: true,
			Hysteresis: 0,
		},
	}

	assertStatus := func(hysteresis int, online bool) {
		if mirror.Status.Hysteresis != hysteresis {
			t.Errorf("updateMirror() failed: hysteresis = %d; want %d\n",
					mirror.Status.Hysteresis, hysteresis)
		}
		if mirror.Status.Online != online {
			t.Errorf("updateMirror() failed: online = %v; want %v\n",
					mirror.Status.Online, online)
		}
	}

	updateMirror("test", mirror, true)
	assertStatus(0, true)
	updateMirror("test", mirror, false)
	assertStatus(1, true)
	updateMirror("test", mirror, false)
	assertStatus(0, false)
	updateMirror("test", mirror, false)
	assertStatus(0, false)
	updateMirror("test", mirror, true)
	assertStatus(1, false)
	updateMirror("test", mirror, false)
	assertStatus(0, false)
	updateMirror("test", mirror, true)
	assertStatus(1, false)
	updateMirror("test", mirror, true)
	assertStatus(0, true)
}
