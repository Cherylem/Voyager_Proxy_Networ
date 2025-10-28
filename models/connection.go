package models

type Connection struct {
	Name     string `json:"name"`
	Server   string `json:"server"`
	Port     int    `json:"port"`
	Protocol string `json:"protocol"`
	Status   string `json:"status"`
	Config   string `json:"config"` // Полная VLESS ссылка
	Country  string `json:"country"`

	// Парсированные параметры
	UUID     string `json:"uuid"`
	Security string `json:"security"`
	Type     string `json:"type"`
	Flow     string `json:"flow"`
	SNI      string `json:"sni"`
	FP       string `json:"fp"`
	PBK      string `json:"pbk"`
	SID      string `json:"sid"`
	SPX      string `json:"spx"`

	Params map[string]string `json:"params"`
}

type ConnectionConfig struct {
	Name   string `json:"name"`
	Config string `json:"config"`
}
