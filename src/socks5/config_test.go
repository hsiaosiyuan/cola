package socks5

import (
	"testing"
)

func TestNewConfig(t *testing.T) {
	var (
		cfg *Config
		err error
	)

	if cfg, err = NewConfig("example_conf.json"); err != nil {
		t.Fatal(err)
	}

	if cfg.ServerPort != 1080 ||
		cfg.UsrPwdPairs["Usr1"] != "Pwd1" ||
		cfg.UsrPwdPairs["Usr2"] != "Pwd2" ||
		cfg.AuthMethods[0] != 0 ||
		cfg.AuthMethods[1] != 2 {
		t.Fatal("Failed to parse")
	}
}
