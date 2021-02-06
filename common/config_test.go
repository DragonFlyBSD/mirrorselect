package common

import "testing"


func TestReadConfig(t *testing.T) {
	fname := "../testdata/mirrorselect.toml"
	cfg := ReadConfig(fname)

	if cfg.Monitor.TLSVerify != false {
		t.Errorf("ReadConfig(%q) failed: TLSVerify = %v, want %v\n",
				fname, cfg.Monitor.TLSVerify, false)
	}

	if len(cfg.Mirrors) != 2 {
		t.Errorf("ReadConfig(%q) failed: got %d mirrors, want %d\n",
				fname, len(cfg.Mirrors), 2)
	}

	for name, mirror := range cfg.Mirrors {
		if mirror.Status.Online != true {
			t.Errorf("ReadConfig(%q) failed: mirror [%s] status != online\n",
					fname, name)
		}
	}
}


func TestResetConfig(t *testing.T) {
	AppConfig.Monitor.TLSVerify = false
	resetConfig()
	if AppConfig.Monitor.TLSVerify != true {
		t.Errorf("resetConfig() failed: TLSVerify = %v, want true\n",
				AppConfig.Monitor.TLSVerify)
	}
}
