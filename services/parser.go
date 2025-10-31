package services

import (
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"
	"vpn-client/models"
)

func ParseVLESS(vlessURL string) (*models.Connection, error) {
	if !strings.HasPrefix(vlessURL, "vless://") {
		return nil, fmt.Errorf("invalid vless URL: must start with 'vless://'")
	}
	if len(vlessURL) < 20 { // Минимальная длина для валидной ссылки
		return nil, fmt.Errorf("invalid vless URL: too short")
	}

	// Убираем префикс vless://
	parts := strings.SplitN(vlessURL[8:], "#", 2)
	mainPart := parts[0]

	var name string
	if len(parts) > 1 {
		name, _ = url.QueryUnescape(parts[1])
	}

	// Разделяем UUID@server:port и параметры
	urlParts := strings.SplitN(mainPart, "?", 2)
	authPart := urlParts[0]
	paramsPart := ""
	if len(urlParts) > 1 {
		paramsPart = urlParts[1]
	}

	// Парсим UUID@server:port
	authParts := strings.SplitN(authPart, "@", 2)
	if len(authParts) != 2 {
		return nil, fmt.Errorf("invalid auth format")
	}

	uuid := authParts[0]
	serverPart := authParts[1]

	// Парсим server:port
	serverParts := strings.SplitN(serverPart, ":", 2)
	if len(serverParts) != 2 {
		return nil, fmt.Errorf("invalid server format")
	}

	server := serverParts[0]
	port, err := strconv.Atoi(serverParts[1])
	if err != nil {
		return nil, fmt.Errorf("invalid port")
	}

	// Парсим параметры
	params, err := url.ParseQuery(paramsPart)
	if err != nil {
		return nil, fmt.Errorf("invalid parameters")
	}

	conn := &models.Connection{
		Name:     name,
		Server:   server,
		Port:     port,
		UUID:     uuid,
		Config:   vlessURL,
		Status:   "disconnected",
		Protocol: "vless",
		Params:   make(map[string]string),
		Country:  "Unknown",
	}
	go func() {
		serverAddr := conn.Server
		// Если server содержит доменное имя, попробуем получить IP
		if net.ParseIP(serverAddr) == nil {
			ips, err := net.LookupIP(serverAddr)
			if err == nil && len(ips) > 0 {
				// выбираем первый IPv4 если есть
				for _, ip := range ips {
					if ip.To4() != nil {
						serverAddr = ip.String()
						break
					}
				}
				// если не нашли IPv4, возьмем первый
				if net.ParseIP(serverAddr) == nil && len(ips) > 0 {
					serverAddr = ips[0].String()
				}
			}
		}

		country, err := GetCountryByIPAPI(serverAddr)
		if err == nil && country != "" && country != "Unknown" {
			conn.Country = country
			// UI обновление можно выполнить через callback если потребуется
		}
	}()
	// Извлекаем основные параметры
	if security := params.Get("security"); security != "" {
		conn.Security = security
	}
	if connType := params.Get("type"); connType != "" {
		conn.Type = connType
	}
	if flow := params.Get("flow"); flow != "" {
		conn.Flow = flow
	}
	if sni := params.Get("sni"); sni != "" {
		conn.SNI = sni
	}
	if fp := params.Get("fp"); fp != "" {
		conn.FP = fp
	}
	if pbk := params.Get("pbk"); pbk != "" {
		conn.PBK = pbk
	}
	if sid := params.Get("sid"); sid != "" {
		conn.SID = sid
	}
	if spx := params.Get("spx"); spx != "" {
		conn.SPX = spx
	}

	// Сохраняем все параметры
	for key, values := range params {
		if len(values) > 0 {
			conn.Params[key] = values[0]
		}
	}

	// Если имя не указано в ссылке, генерируем его
	if conn.Name == "" {
		conn.Name = fmt.Sprintf("%s:%d", conn.Server, conn.Port)
	}

	return conn, nil
}
