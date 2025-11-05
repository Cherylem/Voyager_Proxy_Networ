package services

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"vpn-client/models"
)

func getConfigFilePath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	configDir := filepath.Join(homeDir, "Library", "Application Support", "VoyagerProxyNetwork")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return "", err
	}

	return filepath.Join(configDir, "config.json"), nil
}

type AppConfig struct {
	Connections []models.Connection `json:"connections"`
}

func LoadConfig() (*AppConfig, error) {
	configFile, err := getConfigFilePath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(configFile)
	if err != nil {
		if os.IsNotExist(err) {
			// Файл не существует, создаём пустую конфигурацию
			return &AppConfig{Connections: []models.Connection{}}, nil
		}
		return nil, err
	}

	var config AppConfig
	err = json.Unmarshal(data, &config)
	return &config, err
}

func SaveConfig(config *AppConfig) error {
	configFile, err := getConfigFilePath()
	if err != nil {
		return err
	}

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
