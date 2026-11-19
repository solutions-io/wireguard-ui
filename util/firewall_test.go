package util

import (
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/ngoduykhanh/wireguard-ui/model"
)

func sampleFirewall() model.FirewallSettings {
	return model.FirewallSettings{
		Enabled: true,
		Destinations: []model.FirewallDestination{
			{Address: "49.13.17.136", Description: "winprod.panteia.nl"},
			{Address: " 10.0.0.0/8 ", Description: "private\nnetwork"},
			{Address: "49.13.17.136/32", Description: "duplicate"},
			{Address: "2a01:4f8:c016:58be::/64", Description: "winprod IPv6"},
			{Address: "", Description: "empty rows are ignored"},
		},
		TCPPorts: []string{"443", "80", "8000-8100", "443"},
		UDPPorts: []string{"53"},
	}
}

func TestNormalizeFirewallSettings(t *testing.T) {
	got, err := NormalizeFirewallSettings(sampleFirewall())
	if err != nil {
		t.Fatal(err)
	}
	wantDestinations := []model.FirewallDestination{
		{Address: "49.13.17.136", Description: "winprod.panteia.nl"},
		{Address: "10.0.0.0/8", Description: "private network"},
		{Address: "2a01:4f8:c016:58be::/64", Description: "winprod IPv6"},
	}
	if !reflect.DeepEqual(got.Destinations, wantDestinations) {
		t.Errorf("destinations = %#v, want %#v", got.Destinations, wantDestinations)
	}
	if want := []string{"80", "443", "8000-8100"}; !reflect.DeepEqual(got.TCPPorts, want) {
		t.Errorf("TCP ports = %v, want %v", got.TCPPorts, want)
	}
}

func TestNormalizeFirewallSettingsRejectsInvalidInput(t *testing.T) {
	cases := map[string]model.FirewallSettings{
		"host bits":     {Destinations: []model.FirewallDestination{{Address: "10.0.0.1/8"}}},
		"everything":    {Destinations: []model.FirewallDestination{{Address: "0.0.0.0/0"}}},
		"not an IP":     {Destinations: []model.FirewallDestination{{Address: "example.com"}}},
		"port zero":     {TCPPorts: []string{"0"}},
		"port too high": {UDPPorts: []string{"65536"}},
		"bad range":     {TCPPorts: []string{"9000-8000"}},
		"not a port":    {TCPPorts: []string{"https"}},
	}
	for name, settings := range cases {
		if _, err := NormalizeFirewallSettings(settings); err == nil {
			t.Errorf("%s: expected an error", name)
		}
	}
}

func TestRenderFirewallDisabledOnlyRemovesTable(t *testing.T) {
	rules, err := RenderFirewall(model.FirewallSettings{Enabled: false}, "wg0")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(rules, "delete table inet wireguard_ui") || strings.Contains(rules, "chain forward") {
		t.Errorf("unexpected rules for a disabled firewall:\n%s", rules)
	}
}

func TestRenderFirewallEnabled(t *testing.T) {
	rules, err := RenderFirewall(sampleFirewall(), "wg0")
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		`iifname != "wg0" accept`,
		"49.13.17.136, # winprod.panteia.nl",
		"10.0.0.0/8 # private network",
		"2a01:4f8:c016:58be::/64 # winprod IPv6",
		"8000-8100",
		`counter name "blocked" reject with icmpx admin-prohibited`,
	} {
		if !strings.Contains(rules, want) {
			t.Errorf("rules do not contain %q:\n%s", want, rules)
		}
	}
}

func TestRenderFirewallRejectsBadInterfaceName(t *testing.T) {
	if _, err := RenderFirewall(sampleFirewall(), `wg0" accept`); err == nil {
		t.Error("expected an error for an interface name with quotes")
	}
}

func TestInterfaceNameAndFilePath(t *testing.T) {
	if got := InterfaceNameFromConfigPath("/etc/wireguard/wg0.conf"); got != "wg0" {
		t.Errorf("interface = %q, want wg0", got)
	}
	t.Setenv(FirewallFilePathEnvVar, "")
	if got := FirewallFilePath("/etc/wireguard/wg0.conf"); got != filepath.Join("/etc/wireguard", "wireguard-ui-firewall.nft") {
		t.Errorf("firewall file = %q", got)
	}
}

// TestRenderFirewallIsValidNftables lets nft check the generated rules. It needs nft and the
// NET_ADMIN capability, so it only runs where those are available, such as the test container.
func TestRenderFirewallIsValidNftables(t *testing.T) {
	nft, err := exec.LookPath("nft")
	if err != nil {
		t.Skip("nft is not installed")
	}
	for _, settings := range []model.FirewallSettings{sampleFirewall(), {Enabled: true}, {Enabled: false}} {
		rules, err := RenderFirewall(settings, "wg0")
		if err != nil {
			t.Fatal(err)
		}
		path := filepath.Join(t.TempDir(), "rules.nft")
		if err := os.WriteFile(path, []byte(rules), 0600); err != nil {
			t.Fatal(err)
		}
		out, err := exec.Command(nft, "-c", "-f", path).CombinedOutput()
		if err != nil {
			if strings.Contains(string(out), "Operation not permitted") {
				t.Skip("nft needs the NET_ADMIN capability")
			}
			t.Errorf("nft rejected the rules: %v\n%s\n%s", err, out, rules)
		}
	}
}
