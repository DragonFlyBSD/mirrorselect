package monitor

import (
	"net/url"
	"testing"
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


func TestHysteresis(t *testing.T) {
	// TODO ...
}
