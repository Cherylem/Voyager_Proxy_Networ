package core

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"vpn-client/models"
)

type XrayConfig struct {
	Log       LogConfig     `json:"log"`
	API       APIConfig     `json:"api"`
	Inbounds  []Inbound     `json:"inbounds"`
	Outbounds []Outbound    `json:"outbounds"`
	Stats     StatsConfig   `json:"stats"`
	Policy    PolicyConfig  `json:"policy"`
	Routing   RoutingConfig `json:"routing"`
}

type LogConfig struct {
	Loglevel string `json:"loglevel"`
	Access   string `json:"access"`
	Error    string `json:"error"`
}

type APIConfig struct {
	Tag      string   `json:"tag"`
	Services []string `json:"services"`
}

type Inbound struct {
	Tag      string          `json:"tag"`
	Port     int             `json:"port"`
	Listen   string          `json:"listen"`
	Protocol string          `json:"protocol"`
	Settings json.RawMessage `json:"settings"`
	Sniffing SniffingConfig  `json:"sniffing"`
}

type SniffingConfig struct {
	Enabled      bool     `json:"enabled"`
	DestOverride []string `json:"destOverride"`
}

type Outbound struct {
	Tag            string          `json:"tag"`
	Protocol       string          `json:"protocol"`
	Settings       json.RawMessage `json:"settings"`
	StreamSettings *StreamSettings `json:"streamSettings,omitempty"`
}

type StreamSettings struct {
	Network         string           `json:"network"`
	Security        string           `json:"security"`
	RealitySettings *RealitySettings `json:"realitySettings,omitempty"`
}

type RealitySettings struct {
	Fingerprint  string `json:"fingerprint"`
	ServerName   string `json:"serverName"`
	PublicKey    string `json:"publicKey"`
	ShortId      string `json:"shortId"`
	SpiderX      string `json:"spiderX"`
	MinClientVer string `json:"minClientVer"`
	MaxClientVer string `json:"maxClientVer"`
	MaxTimeDiff  int64  `json:"maxTimeDiff"`
}

type StatsConfig struct{}
type PolicyConfig struct{}

type RoutingConfig struct {
	DomainStrategy string        `json:"domainStrategy"`
	Rules          []RoutingRule `json:"rules"`
}

type RoutingRule struct {
	Type        string   `json:"type"`
	OutboundTag string   `json:"outboundTag"`
	Domain      []string `json:"domain,omitempty"`
	IP          []string `json:"ip,omitempty"`
	Port        string   `json:"port,omitempty"`
}

// GenerateConfig создает конфиг Xray для подключения
func (x *XrayManager) GenerateConfig(conn *models.Connection) (string, error) {
	fmt.Printf("🔧 Debug Connection: Server=%s, Port=%d, UUID=%s, Flow=%s, FP=%s, SNI=%s, PBK=%s, SID=%s\n",
		conn.Server, conn.Port, conn.UUID, conn.Flow, conn.FP, conn.SNI, conn.PBK, conn.SID)

	config := XrayConfig{
		Log: LogConfig{
			Loglevel: "debug",
			Access:   x.getLogPath("access.log"),
			Error:    x.getLogPath("error.log"),
		},
		API: APIConfig{
			Tag:      "api",
			Services: []string{"StatsService", "HandlerService"},
		},
		Inbounds: []Inbound{
			{
				Tag:      "socks-in",
				Port:     1080,
				Listen:   "127.0.0.1",
				Protocol: "socks",
				Settings: json.RawMessage(`{
					"auth": "noauth",
					"udp": true,
					"userLevel": 0
				}`),
				Sniffing: SniffingConfig{
					Enabled:      true,
					DestOverride: []string{"http", "tls", "quic"},
				},
			},
			{
				Tag:      "http-in",
				Port:     1081,
				Listen:   "127.0.0.1",
				Protocol: "http",
				Settings: json.RawMessage(`{
					"timeout": 300,
					"userLevel": 0,
					"allowTransparent": false
				}`),
				Sniffing: SniffingConfig{
					Enabled:      true,
					DestOverride: []string{"http", "tls", "quic"},
				},
			},
		},
		Outbounds: []Outbound{
			{
				Tag:      "proxy",
				Protocol: "vless",
				Settings: json.RawMessage(fmt.Sprintf(`{
					"vnext": [{
						"address": "%s",
						"port": %d,
						"users": [{
							"id": "%s",
							"flow": "%s",
							"encryption": "none",
							"level": 0
						}]
					}]
				}`, conn.Server, conn.Port, conn.UUID, conn.Flow)),
				StreamSettings: &StreamSettings{
					Network:  "tcp",
					Security: "reality",
					RealitySettings: &RealitySettings{
						Fingerprint:  conn.FP,
						ServerName:   conn.SNI,
						PublicKey:    conn.PBK,
						ShortId:      conn.SID,
						SpiderX:      "/",
						MinClientVer: "",
						MaxClientVer: "",
						MaxTimeDiff:  0,
					},
				},
			},
			{
				Tag:      "direct",
				Protocol: "freedom",
				Settings: json.RawMessage(`{
					"domainStrategy": "AsIs"
				}`),
			},
			{
				Tag:      "block",
				Protocol: "blackhole",
				Settings: json.RawMessage(`{
					"response": {
						"type": "http"
					}
				}`),
			},
		},
		Routing: RoutingConfig{
			DomainStrategy: "IPIfNonMatch",
			Rules: []RoutingRule{
				{
					Type:        "field",
					OutboundTag: "direct",
					Domain:      []string{"geosite:cn"},
				},
				{
					Type:        "field",
					OutboundTag: "direct",
					IP:          []string{"geoip:cn", "geoip:private"},
				},
				{
					Type:        "field",
					OutboundTag: "direct",
					Port:        "53",
				},
				{
					Type:        "field",
					OutboundTag: "block",
					Domain:      []string{"geosite:category-ads-all"},
				},
				{
					Type:        "field",
					OutboundTag: "proxy",
					Domain:      []string{"geosite:geolocation-!cn"},
				},
				{
					Type:        "field",
					OutboundTag: "proxy",
					IP:          []string{"0.0.0.0/0", "::/0"},
				},
			},
		},
	}

	// Сохраняем конфиг в файл
	configDir := filepath.Join(x.getXrayDir(), "config")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return "", err
	}

	configPath := filepath.Join(configDir, "config.json")
	file, err := os.Create(configPath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(config); err != nil {
		return "", err
	}

	fmt.Printf("✅ Xray config generated: %s\n", configPath)
	return configPath, nil
}
