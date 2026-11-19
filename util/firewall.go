package util

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/netip"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/labstack/gommon/log"

	"github.com/ngoduykhanh/wireguard-ui/model"
	"github.com/ngoduykhanh/wireguard-ui/store"
)

// FirewallTable is the nftables table that holds the rules for VPN clients. It is separate from
// /etc/nftables.conf: its forward chain runs just before the default one and only rejects traffic.
const FirewallTable = "wireguard_ui"

const (
	FirewallFilePathEnvVar = "WGUI_FIREWALL_FILE_PATH"
	firewallFileName       = "wireguard-ui-firewall.nft"
	maxDescriptionLength   = 100
)

var (
	DefaultFirewallTCPPorts = []string{"53", "80", "443"}
	DefaultFirewallUDPPorts = []string{"53"}

	interfaceNameRegexp = regexp.MustCompile(`^[A-Za-z0-9_.=+-]{1,15}$`)
)

// DefaultFirewallSettings is used when the database has no firewall settings yet.
func DefaultFirewallSettings() model.FirewallSettings {
	return model.FirewallSettings{
		Enabled:      false,
		Destinations: []model.FirewallDestination{},
		TCPPorts:     DefaultFirewallTCPPorts,
		UDPPorts:     DefaultFirewallUDPPorts,
		UpdatedAt:    time.Now().UTC(),
	}
}

// NormalizeFirewallSettings validates the settings and returns them in canonical form:
// destinations without duplicates, ports sorted, descriptions on one line.
func NormalizeFirewallSettings(s model.FirewallSettings) (model.FirewallSettings, error) {
	out := model.FirewallSettings{
		Enabled:      s.Enabled,
		Destinations: []model.FirewallDestination{},
		UpdatedAt:    s.UpdatedAt,
	}
	seen := map[netip.Prefix]bool{}
	for _, d := range s.Destinations {
		address := strings.TrimSpace(d.Address)
		if address == "" {
			continue
		}
		prefix, err := parseDestination(address)
		if err != nil {
			return out, err
		}
		if seen[prefix] {
			continue
		}
		seen[prefix] = true
		out.Destinations = append(out.Destinations, model.FirewallDestination{
			Address:     formatPrefix(prefix),
			Description: cleanDescription(d.Description),
		})
	}

	var err error
	if out.TCPPorts, err = normalizePorts(s.TCPPorts); err != nil {
		return out, fmt.Errorf("TCP ports: %w", err)
	}
	if out.UDPPorts, err = normalizePorts(s.UDPPorts); err != nil {
		return out, fmt.Errorf("UDP ports: %w", err)
	}
	return out, nil
}

func parseDestination(address string) (netip.Prefix, error) {
	var prefix netip.Prefix
	if strings.Contains(address, "/") {
		p, err := netip.ParsePrefix(address)
		if err != nil {
			return prefix, fmt.Errorf("%q is not a valid IP address or network", address)
		}
		if p.Addr().Is4In6() {
			p = netip.PrefixFrom(p.Addr().Unmap(), p.Bits()-96)
		}
		if p != p.Masked() {
			return prefix, fmt.Errorf("%q has host bits set; use %s, or the single address with /32 or /128", address, p.Masked())
		}
		prefix = p
	} else {
		a, err := netip.ParseAddr(address)
		if err != nil {
			return prefix, fmt.Errorf("%q is not a valid IP address or network", address)
		}
		a = a.Unmap()
		prefix = netip.PrefixFrom(a, a.BitLen())
	}
	if prefix.Addr().Zone() != "" {
		return prefix, fmt.Errorf("%q: addresses with a zone are not supported", address)
	}
	if prefix.Bits() == 0 {
		return prefix, fmt.Errorf("%q would allow everything; turn the firewall off instead", address)
	}
	return prefix, nil
}

func formatPrefix(p netip.Prefix) string {
	if p.IsSingleIP() {
		return p.Addr().String()
	}
	return p.String()
}

func cleanDescription(s string) string {
	s = strings.Map(func(r rune) rune {
		if unicode.IsControl(r) {
			return ' '
		}
		return r
	}, s)
	s = strings.TrimSpace(s)
	if r := []rune(s); len(r) > maxDescriptionLength {
		s = string(r[:maxDescriptionLength])
	}
	return s
}

type portRange struct{ from, to int }

func normalizePorts(ports []string) ([]string, error) {
	var ranges []portRange
	seen := map[portRange]bool{}
	for _, p := range ports {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		r, err := parsePortRange(p)
		if err != nil {
			return nil, err
		}
		if !seen[r] {
			seen[r] = true
			ranges = append(ranges, r)
		}
	}
	sort.Slice(ranges, func(i, j int) bool {
		if ranges[i].from != ranges[j].from {
			return ranges[i].from < ranges[j].from
		}
		return ranges[i].to < ranges[j].to
	})
	out := []string{}
	for _, r := range ranges {
		if r.from == r.to {
			out = append(out, strconv.Itoa(r.from))
		} else {
			out = append(out, fmt.Sprintf("%d-%d", r.from, r.to))
		}
	}
	return out, nil
}

func parsePortRange(s string) (portRange, error) {
	from, to, isRange := strings.Cut(s, "-")
	a, errA := parsePort(from)
	b := a
	var errB error
	if isRange {
		b, errB = parsePort(to)
	}
	if errA != nil || errB != nil {
		return portRange{}, fmt.Errorf("%q is not a port (1-65535) or a range such as 8000-8100", s)
	}
	if a > b {
		return portRange{}, fmt.Errorf("%q: the first port must not be higher than the second", s)
	}
	return portRange{a, b}, nil
}

func parsePort(s string) (int, error) {
	n, err := strconv.Atoi(strings.TrimSpace(s))
	if err != nil || n < 1 || n > 65535 {
		return 0, errors.New("invalid port")
	}
	return n, nil
}

// InterfaceNameFromConfigPath returns the interface wg-quick creates for a config file,
// e.g. wg0 for /etc/wireguard/wg0.conf.
func InterfaceNameFromConfigPath(configFilePath string) string {
	return strings.TrimSuffix(filepath.Base(configFilePath), ".conf")
}

// FirewallFilePath is where the generated rules are written: WGUI_FIREWALL_FILE_PATH,
// or a file next to the WireGuard config.
func FirewallFilePath(configFilePath string) string {
	if p := LookupEnvOrString(FirewallFilePathEnvVar, ""); p != "" {
		return p
	}
	return filepath.Join(filepath.Dir(configFilePath), firewallFileName)
}

// RenderFirewall returns the nftables script for the settings. When the firewall is disabled
// the script only removes the table.
func RenderFirewall(s model.FirewallSettings, iface string) (string, error) {
	if !interfaceNameRegexp.MatchString(iface) {
		return "", fmt.Errorf("invalid interface name %q", iface)
	}
	s, err := NormalizeFirewallSettings(s)
	if err != nil {
		return "", err
	}

	var b strings.Builder
	b.WriteString("#!/usr/sbin/nft -f\n")
	b.WriteString("# Generated by wireguard-ui on \"Apply Config\". Do not edit: changes are overwritten.\n\n")
	fmt.Fprintf(&b, "table inet %s\ndelete table inet %s\n", FirewallTable, FirewallTable)
	if !s.Enabled {
		return b.String(), nil
	}

	var v4, v6 []setElement
	for _, d := range s.Destinations {
		prefix, _ := parseDestination(d.Address)
		e := setElement{value: d.Address, comment: d.Description}
		if prefix.Addr().Is4() {
			v4 = append(v4, e)
		} else {
			v6 = append(v6, e)
		}
	}

	fmt.Fprintf(&b, "\ntable inet %s {\n", FirewallTable)
	writeSet(&b, "destinations_v4", "ipv4_addr", v4)
	writeSet(&b, "destinations_v6", "ipv6_addr", v6)
	writeSet(&b, "tcp_ports", "inet_service", portElements(s.TCPPorts))
	writeSet(&b, "udp_ports", "inet_service", portElements(s.UDPPorts))
	b.WriteString("\tcounter blocked {\n\t}\n\n")
	b.WriteString("\tchain forward {\n")
	b.WriteString("\t\ttype filter hook forward priority filter - 10; policy accept;\n")
	fmt.Fprintf(&b, "\t\tiifname != \"%s\" accept\n", iface)
	b.WriteString("\t\tct state established,related accept\n")
	b.WriteString("\t\tip daddr @destinations_v4 accept\n")
	b.WriteString("\t\tip6 daddr @destinations_v6 accept\n")
	b.WriteString("\t\ttcp dport @tcp_ports accept\n")
	b.WriteString("\t\tudp dport @udp_ports accept\n")
	b.WriteString("\t\tmeta l4proto { icmp, ipv6-icmp } accept\n")
	b.WriteString("\t\tcounter name \"blocked\" reject with icmpx admin-prohibited\n")
	b.WriteString("\t}\n}\n")
	return b.String(), nil
}

type setElement struct{ value, comment string }

func portElements(ports []string) []setElement {
	elems := make([]setElement, 0, len(ports))
	for _, p := range ports {
		elems = append(elems, setElement{value: p})
	}
	return elems
}

func writeSet(b *strings.Builder, name, typ string, elems []setElement) {
	fmt.Fprintf(b, "\tset %s {\n\t\ttype %s\n\t\tflags interval\n\t\tauto-merge\n", name, typ)
	if len(elems) > 0 {
		b.WriteString("\t\telements = {\n")
		for i, e := range elems {
			sep := ","
			if i == len(elems)-1 {
				sep = ""
			}
			if e.comment != "" {
				fmt.Fprintf(b, "\t\t\t%s%s # %s\n", e.value, sep, e.comment)
			} else {
				fmt.Fprintf(b, "\t\t\t%s%s\n", e.value, sep)
			}
		}
		b.WriteString("\t\t}\n")
	}
	b.WriteString("\t}\n\n")
}

// ApplyFirewall loads the rules for the VPN interface of configFilePath. They are loaded from a
// temporary file first and only replace the applied file when nft accepts them; nft loads a file
// atomically, so on failure the previous rules stay in place.
func ApplyFirewall(s model.FirewallSettings, configFilePath string) error {
	rules, err := RenderFirewall(s, InterfaceNameFromConfigPath(configFilePath))
	if err != nil {
		return err
	}
	if _, err := exec.LookPath("nft"); err != nil {
		if s.Enabled {
			return errors.New("nft is not installed, so the firewall cannot be loaded")
		}
		return nil
	}
	path := FirewallFilePath(configFilePath)
	tmp := path + ".new"
	if err := os.WriteFile(tmp, []byte(rules), 0600); err != nil {
		return err
	}
	if err := loadFirewallFile(tmp); err != nil {
		os.Remove(tmp)
		return err
	}
	return os.Rename(tmp, path)
}

func loadFirewallFile(path string) error {
	nft, err := exec.LookPath("nft")
	if err != nil {
		return err
	}
	if out, err := exec.Command(nft, "-f", path).CombinedOutput(); err != nil {
		return fmt.Errorf("nft: %v: %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

// firewallFileDefinesTable reports whether the applied rules create the table, i.e. whether the
// firewall was enabled when the config was last applied.
func firewallFileDefinesTable(path string) bool {
	content, err := os.ReadFile(path)
	return err == nil && strings.Contains(string(content), "chain forward")
}

// GetFirewallStatus asks nftables whether the table is loaded and how many packets it rejected.
func GetFirewallStatus() model.FirewallStatus {
	nft, err := exec.LookPath("nft")
	if err != nil {
		return model.FirewallStatus{Error: "nft is not installed on this machine"}
	}
	out, err := exec.Command(nft, "-j", "list", "counter", "inet", FirewallTable, "blocked").Output()
	if err != nil {
		return model.FirewallStatus{}
	}
	var doc struct {
		Nftables []struct {
			Counter *struct {
				Packets uint64 `json:"packets"`
			} `json:"counter"`
		} `json:"nftables"`
	}
	if err := json.Unmarshal(out, &doc); err != nil {
		return model.FirewallStatus{Error: "cannot read the nft output: " + err.Error()}
	}
	for _, item := range doc.Nftables {
		if item.Counter != nil {
			return model.FirewallStatus{Loaded: true, BlockedPackets: item.Counter.Packets}
		}
	}
	return model.FirewallStatus{}
}

// EnsureFirewallLoaded loads the last applied rules again when nftables no longer has them, for
// example after a reboot or "systemctl reload nftables", which flushes the whole ruleset.
// Saved changes that were not applied yet stay unapplied.
func EnsureFirewallLoaded(configFilePath string) {
	path := FirewallFilePath(configFilePath)
	if !firewallFileDefinesTable(path) || GetFirewallStatus().Loaded {
		return
	}
	if err := loadFirewallFile(path); err != nil {
		log.Errorf("The firewall rules are not loaded and could not be loaded from %s: %v", path, err)
		return
	}
	log.Warnf("The firewall rules were not loaded; loaded them from %s", path)
}

// KeepFirewallLoaded runs EnsureFirewallLoaded now and then every interval.
func KeepFirewallLoaded(db store.IStore, interval time.Duration) {
	for {
		if settings, err := db.GetGlobalSettings(); err == nil {
			EnsureFirewallLoaded(settings.ConfigFilePath)
		}
		time.Sleep(interval)
	}
}
