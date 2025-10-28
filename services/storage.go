package services

import (
	"encoding/json"
	"fmt"
	"os"
	"vpn-client/models"
)

const configFile = "configurations/config.json"

type AppConfig struct {
	Connections []models.Connection `json:"connections"`
}

func LoadConfig() (*AppConfig, error) {
	data, err := os.ReadFile(configFile)
	if err != nil {
		if os.IsNotExist(err) {
			// Файл не существует, возвращаем пустую конфигурацию
			return &AppConfig{Connections: []models.Connection{}}, nil
		}
		return nil, err
	}

	var config AppConfig
	err = json.Unmarshal(data, &config)
	return &config, err
}

func SaveConfig(config *AppConfig) error {
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(configFile, data, 0644)
}

func AddConnection(conn models.Connection) error {
	config, err := LoadConfig()
	if err != nil {
		return err
	}

	// Проверка на дубликаты
	for _, existingConn := range config.Connections {
		if existingConn.Config == conn.Config {
			return fmt.Errorf("connection already exists")
		}
		if existingConn.Name == conn.Name {
			return fmt.Errorf("connection with name '%s' already exists", conn.Name)
		}
	}

	config.Connections = append(config.Connections, conn)
	return SaveConfig(config)
}

func GetConnections() ([]models.Connection, error) {
	config, err := LoadConfig()
	if err != nil {
		return nil, err
	}
	return config.Connections, nil
}
