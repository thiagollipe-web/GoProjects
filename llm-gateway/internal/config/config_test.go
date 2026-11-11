package config

import (
	"os"
	"path/filepath"
	"testing"
)

func write(t *testing.T, s string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "c.json")
	if err := os.WriteFile(p, []byte(s), 0o600); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestLoadDefaults(t *testing.T) {
	c, err := Load(write(t, `{"upstreams":[{"name":"a","base_url":"http://x","models":{"m":"m1"}}]}`))
	if err != nil {
		t.Fatal(err)
	}
	if c.Listen != ":8080" || c.BreakerFails != 3 || c.Upstreams[0].Timeout.Seconds() != 60 {
		t.Fatalf("defaults not applied: %+v", c)
	}
}

func TestLoadInvalid(t *testing.T) {
	cases := map[string]string{
		"no upstream": `{}`,
		"no models":   `{"upstreams":[{"name":"a","base_url":"http://x"}]}`,
		"duplicate":   `{"upstreams":[{"name":"a","base_url":"u","models":{"m":"m"}},{"name":"a","base_url":"u","models":{"m":"m"}}]}`,
		"bad timeout": `{"upstreams":[{"name":"a","base_url":"u","models":{"m":"m"},"timeout":5}]}`,
	}
	for name, js := range cases {
		if _, err := Load(write(t, js)); err == nil {
			t.Errorf("%s: expected error", name)
		}
	}
}
