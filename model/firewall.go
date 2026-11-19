package model

import "time"

// FirewallSettings decide which traffic of VPN clients the server forwards.
// Clients may reach the destinations on every port, and anything else only on the listed ports.
type FirewallSettings struct {
	Enabled      bool                  `json:"enabled"`
	Destinations []FirewallDestination `json:"destinations"`
	TCPPorts     []string              `json:"tcp_ports"`
	UDPPorts     []string              `json:"udp_ports"`
	UpdatedAt    time.Time             `json:"updated_at"`
}

// FirewallDestination is an address or network that VPN clients may reach on every port.
type FirewallDestination struct {
	Address     string `json:"address"`
	Description string `json:"description"`
}

// FirewallStatus is what nftables on the server reports about the loaded rules.
type FirewallStatus struct {
	Loaded         bool
	BlockedPackets uint64
	Error          string
}
