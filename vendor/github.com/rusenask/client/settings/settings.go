package settings

import (
	"net"
	"os"
)

func init() {
	Config = InitializeConfiguration()
}

const (
	// EnvRelayAPIAddress allows to override Webhook Relay default API (https://my.webhookrelay.com).
	// This is used by the standalone, self-hosted transponder where it has its own API
	EnvRelayAPIAddress = "RELAY_API_ADDRESS"
	// EnvTunnelServerAddress allows to override tunnel API address where webhookrelayd or relay connect
	// connects to
	EnvTunnelServerAddress     = "RELAY_TUNNEL_API_ADDRESS"
	EnvRelayInsecureSkipVerify = "RELAY_INSECURE"
	EnvRelayRequireTLS         = "RELAY_REQUIRE_TLS"
)

var Config *ConfigManager

var RequireServerTLS = true

// InsecureSkipVerify - skips cert verification when connecting to Webhook Relay
var InsecureSkipVerify = false

// default Webhook Relay configuration
const (
	apiAddress          = "https://my.webhookrelay.com:443"
	tunnelServerIP      = "35.205.239.225"
	tunnelServerAddress = "tnl.webrelay.io:9800" // TCP
	grpcServerAddress   = "my.webhookrelay.com"
	grpcServerPort      = 8080
)

type ConfigManager struct {
	webhookRelayAPIAddress string
	tunnelServerAddress    string
	tunnelServerIPAddress  string

	grpcServerAddress string
	grpcServerPort    int
}

func InitializeConfiguration() *ConfigManager {

	if os.Getenv(EnvRelayInsecureSkipVerify) == "true" {
		InsecureSkipVerify = true
	}
	if os.Getenv(EnvRelayRequireTLS) == "false" {
		RequireServerTLS = false
	}

	if os.Getenv("RELAY_DEV") == "1" {
		return &ConfigManager{
			tunnelServerAddress:    "local-tunnel.webrelay.keel.sh:9400", // needs a hostname to set cert to
			webhookRelayAPIAddress: "https://localhost:9300",
			grpcServerAddress:      "localhost",
			grpcServerPort:         8082,
		}
	}

	config := &ConfigManager{
		webhookRelayAPIAddress: apiAddress,
		tunnelServerAddress:    tunnelServerAddress,
		tunnelServerIPAddress:  tunnelServerIP,
		grpcServerAddress:      grpcServerAddress,
		grpcServerPort:         grpcServerPort,
	}

	// standalone-transponder
	if os.Getenv(EnvRelayAPIAddress) != "" {
		config.webhookRelayAPIAddress = os.Getenv(EnvRelayAPIAddress)
	}
	if os.Getenv(EnvTunnelServerAddress) != "" {
		config.tunnelServerAddress = os.Getenv(EnvTunnelServerAddress)
	}

	return config
}

func (c *ConfigManager) GetWebhookRelayAPIAddress() string {
	return c.webhookRelayAPIAddress
}

func (c *ConfigManager) GetTunnelServerAddress() string {
	return c.tunnelServerAddress
}

func (c *ConfigManager) GetTunnelServerIPAddress() string {
	return c.tunnelServerIPAddress
}

func (c *ConfigManager) SetTunnelServerAddress(addr string) {
	c.tunnelServerAddress = addr
}

func (c *ConfigManager) GetTunnelServerHostname() string {
	host, _, _ := net.SplitHostPort(c.tunnelServerAddress)
	return host
}

func (c *ConfigManager) GetGRPCServerAddress() string {
	return c.grpcServerAddress
}

func (c *ConfigManager) GetGRPCServerPort() int {
	return c.grpcServerPort
}

// UpdatesURL - specifies endpoint for self updates
const UpdatesURL = "https://storage.googleapis.com/webhookrelay/updates/"

// IngressStableRBACManifest - stable ingress install manifest with rbac
const IngressStableRBACManifest = "https://raw.githubusercontent.com/webrelay/ingress/master/deployment/deployment-rbac.yaml"

// IngressStableNoRBACManifest - stable ingress install manifest without rbac
const IngressStableNoRBACManifest = "https://raw.githubusercontent.com/webrelay/ingress/master/deployment/deployment-norbac.yaml"
