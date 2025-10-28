package services

import (
	"fmt"
	"io"
	"net/http"
	"time"
)

// Используем ip-api.com (бесплатно, без лимитов для некоммерческого использования)
//func GetCountryByIPAPI(ip string) (string, error) {
//	if ip == "" {
//		return "Unknown", nil
//	}
//
//	client := &http.Client{Timeout: 10 * time.Second}
//	url := fmt.Sprintf("http://ip-api.com/json/%s?fields=country", ip)
//
//	resp, err := client.Get(url)
//	if err != nil {
//		return "", err
//	}
//	defer resp.Body.Close()
//
//	if resp.StatusCode != http.StatusOK {
//		return "", fmt.Errorf("API returned status: %d", resp.StatusCode)
//	}
//
//	var result struct {
//		Country string `json:"country"`
//		Status  string `json:"status"`
//	}
//
//	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
//		return "", err
//	}
//
//	if result.Status != "success" {
//		return "Unknown", nil
//	}
//
//	return result.Country, nil
//}

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
