package core

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
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
	Protocol    []string `json:"protocol,omitempty"`
}

// GenerateConfig создает конфиг Xray для подключения
func (x *XrayManager) GenerateConfig(conn *models.Connection) (string, error) {
	// Сначала вычислим путь куда будет записан config.json — whitelist будет искаться рядом с ним
	configDir := filepath.Join(x.getXrayDir(), "config")

	// Загружаем whitelist домены (если есть) из той же директории, где будет config.json
	whitelistPath := filepath.Join(configDir, "whitelist.json")
	whitelist := x.loadWhitelistDomains(whitelistPath)

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
				Port:     x.socksPort,
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
				Port:     x.httpPort,
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
		Outbounds: func() []Outbound {
			proxySettings := json.RawMessage(fmt.Sprintf(`{
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
				}`, conn.Server, conn.Port, conn.UUID, conn.Flow))

			proxyOutbound := Outbound{
				Tag:      "proxy",
				Protocol: "vless",
				Settings: proxySettings,
			}

			if conn.Security == "reality" || (conn.PBK != "" && conn.SID != "") {
				proxyOutbound.StreamSettings = &StreamSettings{
					Network:  "tcp",
					Security: "reality",
					RealitySettings: &RealitySettings{
						Fingerprint:  conn.FP,
						ServerName:   conn.SNI,
						PublicKey:    conn.PBK,
						ShortId:      conn.SID,
						SpiderX:      conn.SPX,
						MinClientVer: "",
						MaxClientVer: "",
						MaxTimeDiff:  0,
					},
				}
			} else {
				proxyOutbound.StreamSettings = &StreamSettings{
					Network:  "tcp",
					Security: "none",
				}
			}

			return []Outbound{
				proxyOutbound,
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
			}
		}(),
		Routing: RoutingConfig{
			DomainStrategy: "IPIfNonMatch",
			Rules: func() []RoutingRule {
				base := []RoutingRule{
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
					OutboundTag: "block",
					Domain:      []string{"geosite:category-ads-all"},
				},
				{
					Type:        "field",
					OutboundTag: "block",
					Port:        "3478,3479,5349,5350",
				},
				{
					Type:        "field",
					OutboundTag: "block",
					Protocol:    []string{"bittorrent"},
				},
				{
					Type:        "field",
					OutboundTag: "proxy",
					Port:        "53",
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
				}


				if len(whitelist) > 0 {
					wlRule := RoutingRule{
						Type:        "field",
						OutboundTag: "direct",
						Domain:      whitelist,
					}

					base = append([]RoutingRule{wlRule}, base...)
				}
				return base
			}(),
		},
	}

	// Сохраняем конфиг в файл
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

	fmt.Printf("✅ Xray config generated with DNS/WebRTC leak protection: %s\n", configPath)
	return configPath, nil
}


func (x *XrayManager) loadWhitelistDomains(preferredPath string) []string {

	candidates := []string{}
	if preferredPath != "" {
		candidates = append(candidates, preferredPath)
	}

	binDir := filepath.Dir(os.Args[0])
	candidates = append(candidates, filepath.Join(binDir, "configurations", "whitelist.json"))

	candidates = append(candidates, "configurations/whitelist.json")

	var found string
	for _, p := range candidates {
		if p == "" {
			continue
		}
		if _, err := os.Stat(p); err == nil {
			found = p
			break
		}
	}
	if found == "" {
		return nil
	}

	data, err := os.ReadFile(found)
	if err != nil {
		return nil
	}

	var raw []string
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil
	}

	out := make([]string, 0, len(raw))
	seen := make(map[string]bool)
	for _, entry := range raw {
		s := strings.TrimSpace(entry)
		if s == "" {
			continue
		}
		// Попробуем распарсить как URL
		if u, err := url.Parse(s); err == nil && u.Host != "" {
			s = u.Host
		} else {
			// убираем схему, слеши и префиксы вручную
			s = strings.TrimPrefix(s, "https://")
			s = strings.TrimPrefix(s, "http://")
			s = strings.TrimSuffix(s, "/")
		}
		// Обрезаем порт, если есть
		if idx := strings.Index(s, ":"); idx != -1 {
			s = s[:idx]
		}
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		dom := "domain:" + s
		if !seen[dom] {
			seen[dom] = true
			out = append(out, dom)
		}
	}
	return out
}

