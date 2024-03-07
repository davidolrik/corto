package cmd

import (
	"testing"

	"github.com/spf13/viper"
)

func TestServerAddr(t *testing.T) {
	viper.Set("server.ip", "127.0.0.1")
	viper.Set("server.port", 3000)
	t.Cleanup(func() {
		viper.Set("server.ip", nil)
		viper.Set("server.port", nil)
	})

	if got := serverAddr(); got != "127.0.0.1:3000" {
		t.Errorf("expected addr %q, got %q", "127.0.0.1:3000", got)
	}

	viper.Set("server.ip", "0.0.0.0")
	if got := serverAddr(); got != "0.0.0.0:3000" {
		t.Errorf("expected addr %q, got %q", "0.0.0.0:3000", got)
	}
}
