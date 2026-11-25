// SPDX-License-Identifier: BSD-3-Clause
//
// Copyright (c) 2021-2026 The DragonFly Project.

package monitor

import (
	"bufio"
	"encoding/pem"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/DragonFlyBSD/mirrorselect/common"
)

func writeCertPEM(t *testing.T, der []byte) string {
	t.Helper()

	fname := filepath.Join(t.TempDir(), "cert.pem")
	data := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
	if err := os.WriteFile(fname, data, 0644); err != nil {
		t.Fatalf("Failed to write certificate: %v", err)
	}
	return fname
}

func closedTcpURL(t *testing.T, scheme string) string {
	t.Helper()

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Failed to listen: %v", err)
	}
	addr := ln.Addr().String()
	if err := ln.Close(); err != nil {
		t.Fatalf("Failed to close listener: %v", err)
	}
	return scheme + "://" + addr
}

func startTestFtpServer(t *testing.T) string {
	t.Helper()

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Failed to listen: %v", err)
	}
	t.Cleanup(func() {
		ln.Close()
	})

	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			go serveTestFtpConn(conn)
		}
	}()

	return "ftp://" + ln.Addr().String()
}

// Implement just enough of the FTP protocol for ftpCheck.
func serveTestFtpConn(conn net.Conn) {
	defer conn.Close()

	fmt.Fprint(conn, "220 Test FTP server ready.\r\n")
	reader := bufio.NewReader(conn)
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			return
		}

		fields := strings.SplitN(strings.TrimSpace(line), " ", 2)
		switch strings.ToUpper(fields[0]) {
		case "USER":
			fmt.Fprint(conn, "331 Password required.\r\n")
		case "PASS":
			fmt.Fprint(conn, "230 Logged in.\r\n")
		case "FEAT":
			fmt.Fprint(conn, "211 End\r\n")
		case "TYPE":
			fmt.Fprint(conn, "200 Type set.\r\n")
		case "CWD":
			if len(fields) != 2 || strings.Contains(fields[1], "missing") {
				fmt.Fprint(conn, "550 No such directory.\r\n")
				return
			}
			fmt.Fprint(conn, "250 Directory changed.\r\n")
		case "QUIT":
			fmt.Fprint(conn, "221 Bye.\r\n")
			return
		default:
			fmt.Fprint(conn, "500 Command not implemented.\r\n")
		}
	}
}

func TestHttpCheck(t *testing.T) {
	oldTLSVerify := appConfig.Monitor.TLSVerify
	oldTimeout := appConfig.Monitor.Timeout
	t.Cleanup(func() {
		appConfig.Monitor.TLSVerify = oldTLSVerify
		appConfig.Monitor.Timeout = oldTimeout
	})
	appConfig.Monitor.Timeout = 2

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/missing" {
			http.NotFound(w, r)
			return
		}
		w.WriteHeader(http.StatusOK)
	})
	httpSrv := httptest.NewServer(handler)
	defer httpSrv.Close()
	tlsSrv := httptest.NewTLSServer(handler)
	defer tlsSrv.Close()

	// Trust the test server's self-signed certificate when verifying TLS.
	t.Setenv("SSL_CERT_FILE", writeCertPEM(t, tlsSrv.Certificate().Raw))

	okURLs := []string{
		httpSrv.URL + "/",
		tlsSrv.URL + "/",
		httpSrv.URL + "/BSD/DragonFlyBSD/dports",
	}
	for _, tlsVerify := range []bool{true, false} {
		appConfig.Monitor.TLSVerify = tlsVerify
		for _, utext := range okURLs {
			u, _ := url.Parse(utext)
			status, err := httpCheck(u)
			if err != nil || !status {
				t.Errorf("httpCheck(%q) = (%v, %v); want %v\n",
					u, status, err, true)
			}
		}
	}

	failURLs := []string{
		closedTcpURL(t, "http") + "/",
		httpSrv.URL + "/missing",
	}
	for _, utext := range failURLs {
		u, _ := url.Parse(utext)
		status, err := httpCheck(u)
		if err == nil || status {
			t.Errorf("httpCheck(%q) = (%v, %v); want %v\n",
				u, status, err, false)
		}
	}

	invalidURLs := []string{
		"ftp://127.0.0.1/",
		"xxx://www.example.com/",
	}
	for _, utext := range invalidURLs {
		u, _ := url.Parse(utext)
		status, err := httpCheck(u)
		if err == nil || status {
			t.Errorf("httpCheck(%q) = (%v, %v); want %v\n",
				u, status, err, false)
		}
	}
}

func TestFtpCheck(t *testing.T) {
	oldTimeout := appConfig.Monitor.Timeout
	t.Cleanup(func() {
		appConfig.Monitor.Timeout = oldTimeout
	})
	appConfig.Monitor.Timeout = 2

	ftpURL := startTestFtpServer(t)
	okURLs := []string{
		ftpURL + "/",
		ftpURL + "/pub/FreeBSD/",
	}
	for _, utext := range okURLs {
		u, _ := url.Parse(utext)
		status, err := ftpCheck(u)
		if err != nil || !status {
			t.Errorf("ftpCheck(%q) = (%v, %v); want %v\n",
				u, status, err, true)
		}
	}

	failURLs := []string{
		closedTcpURL(t, "ftp") + "/",
		ftpURL + "/missing/",
	}
	for _, utext := range failURLs {
		u, _ := url.Parse(utext)
		status, err := ftpCheck(u)
		if err == nil || status {
			t.Errorf("ftpCheck(%q) = (%v, %v); want %v\n",
				u, status, err, false)
		}
	}

	invalidURLs := []string{
		"http://127.0.0.1/",
		"xxx://www.example.com/",
	}
	for _, utext := range invalidURLs {
		u, _ := url.Parse(utext)
		status, err := ftpCheck(u)
		if err == nil || status {
			t.Errorf("ftpCheck(%q) = (%v, %v); want %v\n",
				u, status, err, false)
		}
	}
}

func TestHysteresis(t *testing.T) {
	oldHysteresis := appConfig.Monitor.Hysteresis
	t.Cleanup(func() {
		appConfig.Monitor.Hysteresis = oldHysteresis
	})
	appConfig.Monitor.Hysteresis = 2

	mirror := &common.Mirror{
		Name: "Test",
		Status: common.MirrorStatus{
			Online:     true,
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
