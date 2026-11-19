package util

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ngoduykhanh/wireguard-ui/model"
)

func TestWriteWireGuardServerConfigKeepsFreeTextOnOneLine(t *testing.T) {
	dir := t.TempDir()
	settings := model.GlobalSetting{ConfigFilePath: filepath.Join(dir, "wg0.conf")}
	server := model.Server{
		Interface: &model.ServerInterface{Addresses: []string{"10.99.0.1/24"}, ListenPort: 51820},
		KeyPair:   &model.ServerKeypair{PrivateKey: "server-private-key"},
	}
	injected := "x\n[Interface]\nPostUp = touch /tmp/pwned"
	clients := []model.ClientData{{Client: &model.Client{
		Name:            injected,
		Email:           injected,
		Endpoint:        injected,
		AdditionalNotes: "a\r\nPostUp = touch /tmp/pwned",
		AllocatedIPs:    []string{"10.99.0.2/32"},
		PublicKey:       "client-public-key",
		Enabled:         true,
	}}}

	if err := WriteWireGuardServerConfig(os.DirFS("../templates"), server, clients, nil, settings); err != nil {
		t.Fatal(err)
	}
	content, err := os.ReadFile(settings.ConfigFilePath)
	if err != nil {
		t.Fatal(err)
	}

	interfaces := 0
	for _, line := range strings.Split(string(content), "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "[Interface]" {
			interfaces++
		}
		if strings.HasPrefix(trimmed, "PostUp") && strings.TrimSpace(strings.TrimPrefix(trimmed, "PostUp =")) != "" {
			t.Errorf("injected directive in config: %q", line)
		}
	}
	if interfaces != 1 {
		t.Errorf("want 1 [Interface] section, got %d:\n%s", interfaces, content)
	}
}

func TestValidateEndpoint(t *testing.T) {
	valid := []string{"", "vpn.example.com:51820", "1.2.3.4:51820", "[2001:db8::1]:51820"}
	invalid := []string{"vpn.example.com", "1.2.3.4:0", "1.2.3.4:70000", ":51820", "a b:1", "x:1\nPostUp = id"}
	for _, e := range valid {
		if !ValidateEndpoint(e) {
			t.Errorf("ValidateEndpoint(%q) = false, want true", e)
		}
	}
	for _, e := range invalid {
		if ValidateEndpoint(e) {
			t.Errorf("ValidateEndpoint(%q) = true, want false", e)
		}
	}
}

func TestHasControlChars(t *testing.T) {
	if HasControlChars("Danny Kleine") {
		t.Error("plain name reported as containing control characters")
	}
	for _, s := range []string{"a\nb", "a\rb", "a\x00b", "a\tb"} {
		if !HasControlChars(s) {
			t.Errorf("HasControlChars(%q) = false, want true", s)
		}
	}
}
