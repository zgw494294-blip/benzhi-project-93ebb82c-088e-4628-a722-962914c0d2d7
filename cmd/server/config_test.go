package main

import "testing"

func TestValidateAddress(t *testing.T) {
	for _, valid := range []string{"127.0.0.1:19081", "127.0.0.2:65535", "[::1]:19081"} {
		if err := validateAddress(valid); err != nil {
			t.Errorf("%s should be valid: %v", valid, err)
		}
	}
	for _, invalid := range []string{"0.0.0.0:19081", ":19081", "127.0.0.1:0", "localhost:19081", "127.0.0.1:70000"} {
		if err := validateAddress(invalid); err == nil {
			t.Errorf("%s should be rejected", invalid)
		}
	}
}

func TestDefaultAddressIsHighLoopbackPort(t *testing.T) {
	cfg, err := parseConfig(nil)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.addr != "127.0.0.1:19081" {
		t.Fatalf("addr=%s", cfg.addr)
	}
}
