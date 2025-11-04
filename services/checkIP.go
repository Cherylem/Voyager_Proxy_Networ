package services

import (
	"fmt"
	"io"
	"net/http"
	"time"
)

func GetCountryByIPAPI(ip string) (string, error) {


	if ip == "" {
		return "Unknown", nil
	}

	client := &http.Client{Timeout: 10 * time.Second}

	// Попробуем ipapi.co
	url := fmt.Sprintf("https://ipapi.co/%s/country_name/", ip)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", err
	}

	req.Header.Set("User-Agent", "VPN-Client/1.0")

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("ipapi.co returned status: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	country := string(body)
	if country == "" {
		return "Unknown", nil
	}
	
	return country, nil
}
