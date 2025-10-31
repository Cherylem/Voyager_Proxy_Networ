package core

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"strconv"
	"sync"
	"time"

	"vpn-client/models"
)

type XrayManager struct {
	process       *os.Process
	configPath    string
	xrayPath      string
	isRunning     bool
	stats         *XrayStats
	statsMu       sync.RWMutex
	restartCount  int
	currentConfig *models.Connection
}

type XrayStats struct {
	UploadBytes     int64     `json:"upload"`
	DownloadBytes   int64     `json:"download"`
	Ping            int       `json:"ping"`
	ConnectionCount int       `json:"connectionCount"`
	LastUpdate      time.Time `json:"lastUpdate"`
}

func NewXrayManager() *XrayManager {
	return &XrayManager{
		stats: &XrayStats{},
	}
}

// EnsureXrayBinary проверяет и скачивает Xray если нужно
func (x *XrayManager) EnsureXrayBinary() error {
	xrayDir := x.getXrayDir()
	if err := os.MkdirAll(xrayDir, 0755); err != nil {
		return fmt.Errorf("failed to create xray directory: %v", err)
	}

	x.xrayPath = filepath.Join(xrayDir, "xray")
	if runtime.GOOS == "windows" {
		x.xrayPath += ".exe"
	}

	// Проверяем существует ли бинарник
	if _, err := os.Stat(x.xrayPath); os.IsNotExist(err) {
		fmt.Println("📥 Xray binary not found, downloading...")
		if err := x.downloadXray(); err != nil {
			return fmt.Errorf("failed to download xray: %v", err)
		}
	}

	// Скачиваем geoip и geosite файлы если нужно
	if err := x.downloadGeoData(); err != nil {
		fmt.Printf("⚠️ Failed to download geo data: %v\n", err)
	}

	// Проверяем что бинарник исполняемый
	if err := os.Chmod(x.xrayPath, 0755); err != nil {
		return fmt.Errorf("failed to make xray executable: %v", err)
	}

	fmt.Printf("✅ Xray binary ready: %s\n", x.xrayPath)
	return nil
}

// downloadGeoData скачивает geoip и geosite файлы
func (x *XrayManager) downloadGeoData() error {
	geoDir := filepath.Join(x.getXrayDir(), "geoip")
	if err := os.MkdirAll(geoDir, 0755); err != nil {
		return err
	}

	geoFiles := map[string]string{
		"geoip.dat":   "https://github.com/v2fly/geoip/releases/latest/download/geoip.dat",
		"geosite.dat": "https://github.com/v2fly/domain-list-community/releases/latest/download/dlc.dat",
	}

	for filename, url := range geoFiles {
		filePath := filepath.Join(geoDir, filename)
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			fmt.Printf("📥 Downloading %s...\n", filename)

			resp, err := http.Get(url)
			if err != nil {
				return fmt.Errorf("failed to download %s: %v", filename, err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				return fmt.Errorf("download %s failed with status: %d", filename, resp.StatusCode)
			}

			file, err := os.Create(filePath)
			if err != nil {
				return err
			}
			defer file.Close()

			if _, err := io.Copy(file, resp.Body); err != nil {
				return err
			}

			fmt.Printf("✅ %s downloaded\n", filename)
		}
	}

	return nil
}

// getXrayDir возвращает путь к директории с Xray
func (x *XrayManager) getXrayDir() string {
	configDir, err := os.UserConfigDir()
	if err != nil {
		configDir = "."
	}
	return filepath.Join(configDir, "vpn-client", "xray")
}

// downloadXray скачивает Xray для текущей платформы
func (x *XrayManager) downloadXray() error {
	var downloadURL string

	switch {
	case runtime.GOOS == "darwin" && runtime.GOARCH == "arm64":
		downloadURL = "https://github.com/XTLS/Xray-core/releases/download/v1.8.4/Xray-macos-arm64-v8a.zip"
	case runtime.GOOS == "darwin" && runtime.GOARCH == "amd64":
		downloadURL = "https://github.com/XTLS/Xray-core/releases/download/v1.8.4/Xray-macos-64.zip"
	case runtime.GOOS == "windows":
		downloadURL = "https://github.com/XTLS/Xray-core/releases/download/v1.8.4/Xray-windows-64.zip"
	case runtime.GOOS == "linux" && runtime.GOARCH == "amd64":
		downloadURL = "https://github.com/XTLS/Xray-core/releases/download/v1.8.4/Xray-linux-64.zip"
	default:
		return fmt.Errorf("unsupported platform: %s/%s", runtime.GOOS, runtime.GOARCH)
	}

	fmt.Printf("📥 Downloading Xray from: %s\n", downloadURL)

	// Создаем временный файл
	tmpFile, err := os.CreateTemp("", "xray-download-*.zip")
	if err != nil {
		return err
	}
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()

	// Скачиваем
	resp, err := http.Get(downloadURL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed with status: %d", resp.StatusCode)
	}

	// Копируем данные
	if _, err := io.Copy(tmpFile, resp.Body); err != nil {
		return err
	}

	// Распаковываем
	cmd := exec.Command("unzip", "-o", tmpFile.Name(), "-d", filepath.Dir(x.xrayPath))
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("unzip failed: %v, output: %s", err, output)
	}

	fmt.Println("✅ Xray downloaded and extracted")
	return nil
}

// Start запускает Xray с мониторингом
func (x *XrayManager) Start(conn *models.Connection) error {
	if x.isRunning {
		return fmt.Errorf("xray is already running")
	}

	fmt.Println("🚀 Starting Xray...")
	x.currentConfig = conn

	// 1. Убедимся что бинарник есть
	if err := x.EnsureXrayBinary(); err != nil {
		return fmt.Errorf("xray binary not available: %v", err)
	}

	// 2. Генерируем конфиг
	configPath, err := x.GenerateConfig(conn)
	if err != nil {
		return fmt.Errorf("failed to generate config: %v", err)
	}
	x.configPath = configPath

	// 3. Проверим существует ли конфиг файл
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return fmt.Errorf("config file does not exist: %s", configPath)
	}

	// 4. Прочитаем и выведем конфиг для отладки
	configContent, _ := os.ReadFile(configPath)
	fmt.Printf("📋 Config file content (%d bytes):\n%s\n", len(configContent), string(configContent))

	// 5. Запускаем процесс с минимальными аргументами
	fmt.Printf("🔧 Executing: %s -config %s\n", x.xrayPath, configPath)
	cmd := exec.Command(x.xrayPath, "-config", configPath)

	// 6. Создаем пайпы для реального времени чтения логов
	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		return err
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start xray: %v", err)
	}

	x.process = cmd.Process
	x.isRunning = true

	fmt.Printf("✅ Xray started with PID: %d\n", cmd.Process.Pid)

	// 7. Читаем логи в реальном времени
	go x.readLogs(stdoutPipe, stderrPipe)

	// 8. Проверяем что процесс запустился и порты слушают
	time.Sleep(2 * time.Second)
	x.checkPorts()

	// 9. Настраиваем системный прокси
	if err := x.SetupSystemProxy(); err != nil {
		fmt.Printf("⚠️ Failed to setup system proxy: %v\n", err)
	} else {
		fmt.Println("✅ System proxy configured")
	}

	// 10. Проверяем соединение
	go func() {
		time.Sleep(3 * time.Second)
		if x.TestConnection() {
			fmt.Println("🎉 VPN connection is working!")
		} else {
			fmt.Println("❌ VPN connection test failed")
		}
	}()

	// Запускаем отдельную горутину для ожидания процесса
	go x.monitorProcess(cmd)

	// 11. Запускаем мониторинг
	go x.monitorStats()
	go x.healthCheck()

	return nil
}

// monitorProcess отслеживает состояние процесса
func (x *XrayManager) monitorProcess(cmd *exec.Cmd) {
	state, err := cmd.Process.Wait()
	if err != nil {
		fmt.Printf("❌ Error waiting for process: %v\n", err)
	}
	x.isRunning = false

	// Очищаем системный прокс и при выходе
	x.ClearSystemProxy()

	fmt.Printf("⚠️ Xray process exited with code: %d\n", state.ExitCode())

	// Автоперезапуск
	if x.restartCount < 3 && x.currentConfig != nil {
		fmt.Printf("🔄 Auto-restarting Xray (attempt %d)...\n", x.restartCount+1)
		x.restartCount++
		time.Sleep(2 * time.Second)
		x.Start(x.currentConfig)
	} else {
		fmt.Println("🛑 Auto-restart disabled")
	}
}

// SetupSystemProxy настраивает системный прокси
func (x *XrayManager) SetupSystemProxy() error {
	switch runtime.GOOS {
	case "windows":
		return x.setupWindowsProxy()
	case "darwin":
		return x.setupMacOSProxy()
	case "linux":
		fmt.Println("ℹ️ Linux: Please configure system proxy manually or use browser extensions")
		return nil
	default:
		return fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}
}

func (x *XrayManager) setupWindowsProxy() error {
	// Настройка HTTP прокси
	cmd := exec.Command("netsh", "winhttp", "set", "proxy", "127.0.0.1:1081")
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to set windows proxy: %v, output: %s", err, output)
	}
	fmt.Println("✅ Windows system proxy configured")
	return nil
}

func (x *XrayManager) setupMacOSProxy() error {
	// Получаем список сетевых сервисов
	networks, err := exec.Command("networksetup", "-listallnetworkservices").Output()
	if err != nil {
		return err
	}

	scanner := bufio.NewScanner(bytes.NewReader(networks))
	for scanner.Scan() {
		service := strings.TrimSpace(scanner.Text())
		if service != "" && !strings.Contains(service, "*") {
			// Настройка HTTP прокси
			exec.Command("networksetup", "-setwebproxy", service, "127.0.0.1", "1081").Run()
			// Настройка HTTPS прокси
			exec.Command("networksetup", "-setsecurewebproxy", service, "127.0.0.1", "1081").Run()
			// Настройка SOCKS прокси
			exec.Command("networksetup", "-setsocksfirewallproxy", service, "127.0.0.1", "1080").Run()

			// Включаем прокси
			exec.Command("networksetup", "-setwebproxystate", service, "on").Run()
			exec.Command("networksetup", "-setsecurewebproxystate", service, "on").Run()
			exec.Command("networksetup", "-setsocksfirewallproxystate", service, "on").Run()
		}
	}

	if err := scanner.Err(); err != nil {
		return err
	}

	fmt.Println("✅ macOS system proxy configured")
	return nil
}

// ClearSystemProxy очищает настройки прокси
func (x *XrayManager) ClearSystemProxy() error {
	switch runtime.GOOS {
	case "windows":
		cmd := exec.Command("netsh", "winhttp", "reset", "proxy")
		if output, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("failed to reset windows proxy: %v, output: %s", err, output)
		}
	case "darwin":
		networks, err := exec.Command("networksetup", "-listallnetworkservices").Output()
		if err != nil {
			return err
		}

		scanner := bufio.NewScanner(bytes.NewReader(networks))
		for scanner.Scan() {
			service := strings.TrimSpace(scanner.Text())
			if service != "" && !strings.Contains(service, "*") {
				exec.Command("networksetup", "-setwebproxystate", service, "off").Run()
				exec.Command("networksetup", "-setsecurewebproxystate", service, "off").Run()
				exec.Command("networksetup", "-setsocksfirewallproxystate", service, "off").Run()
			}
		}
	}
	fmt.Println("✅ System proxy cleared")
	return nil
}

// TestConnection проверяет что прокси работает
func (x *XrayManager) TestConnection() bool {
	proxyURL, err := url.Parse("http://127.0.0.1:1081")
	if err != nil {
		fmt.Printf("❌ Failed to parse proxy URL: %v\n", err)
		return false
	}

	client := &http.Client{
		Transport: &http.Transport{
			Proxy: http.ProxyURL(proxyURL),
		},
		Timeout: 10 * time.Second,
	}

	// Тестируем через несколько сервисов
	testURLs := []string{
		"http://httpbin.org/ip",
		"http://api.ipify.org?format=json",
		"http://ifconfig.me/ip",
	}

	for _, testURL := range testURLs {
		resp, err := client.Get(testURL)
		if err != nil {
			fmt.Printf("❌ Proxy test failed for %s: %v\n", testURL, err)
			continue
		}
		defer resp.Body.Close()

		if resp.StatusCode == 200 {
			body, _ := io.ReadAll(resp.Body)
			fmt.Printf("✅ Proxy test successful (%s): %s\n", testURL, string(body))
			return true
		}
	}

	return false
}

// readLogs читает логи Xray в реальном времени
func (x *XrayManager) readLogs(stdout, stderr io.Reader) {
	// Читаем stdout
	go func() {
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			line := scanner.Text()
			fmt.Printf("🔵 XRAY: %s\n", line)
		}
	}()

	// Читаем stderr
	go func() {
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			line := scanner.Text()
			fmt.Printf("🔴 XRAY ERROR: %s\n", line)
		}
	}()
}

// checkPorts проверяет что порты слушают
func (x *XrayManager) checkPorts() {
	ports := []int{1080, 1081}
	for _, port := range ports {
		if x.checkPortListening(port) {
			fmt.Printf("✅ PORT %d is listening\n", port)
		} else {
			fmt.Printf("❌ PORT %d is NOT listening\n", port)
		}
	}
}

func (x *XrayManager) checkPortListening(port int) bool {
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%d", port), 1*time.Second)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}

// monitorStats собирает статистику из Xray API
func (x *XrayManager) monitorStats() {
	for x.isRunning {
		stats, err := x.getStats()
		if err == nil && stats != nil {
			x.statsMu.Lock()
			x.stats = stats
			x.stats.LastUpdate = time.Now()
			x.statsMu.Unlock()
			// Debug: print received snapshot
			fmt.Printf("[monitorStats] snapshot: upload=%d download=%d ping=%d connections=%d\n",
				stats.UploadBytes, stats.DownloadBytes, stats.Ping, stats.ConnectionCount)
		}
		if err != nil {
			fmt.Printf("[monitorStats] getStats error: %v\n", err)
		}
		time.Sleep(1 * time.Second)
	}
}

// getStats получает статистику через Xray API
func (x *XrayManager) getStats() (*XrayStats, error) {
	// Попытка получить статистику через локальный Xray API (если он слушает)
	apiCandidates := []string{
		"http://127.0.0.1:10085/stats",
		"http://127.0.0.1:10086/stats",
		"http://127.0.0.1:8080/stats",
	}

	client := &http.Client{Timeout: 3 * time.Second}
	for _, u := range apiCandidates {
		resp, err := client.Get(u)
		if err != nil {
			continue
		}
		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil || resp.StatusCode != http.StatusOK {
			continue
		}

		// Debug: print HTTP body (truncated) so we can see exact format returned by admin API
		bodyStr := string(body)
		if len(bodyStr) > 2000 {
			bodyStr = bodyStr[:2000] + "...(truncated)"
		}
		fmt.Printf("[getStats] HTTP %s -> %s\n", u, bodyStr)

		// Пытаемся распарсить JSON в нескольких вариантах: простой ключ-число
		var parsed interface{}
		if err := json.Unmarshal(body, &parsed); err == nil {
			// рекурсивный поиск числовых полей по подстроке ключа
			var findNumber func(interface{}, string) (int64, bool)
			findNumber = func(v interface{}, substr string) (int64, bool) {
				switch t := v.(type) {
				case map[string]interface{}:
					for k, val := range t {
						if strings.Contains(strings.ToLower(k), substr) {
							switch nv := val.(type) {
							case float64:
								return int64(nv), true
							case string:
								if n, err := strconv.ParseInt(nv, 10, 64); err == nil {
									return n, true
								}
							}
						}
						if res, ok := findNumber(val, substr); ok {
							return res, true
						}
					}
				case []interface{}:
					for _, item := range t {
						if res, ok := findNumber(item, substr); ok {
							return res, true
						}
					}
				}
				return 0, false
			}

			stats := &XrayStats{}
			if v, ok := findNumber(parsed, "upload"); ok {
				stats.UploadBytes = v
			}
			if v, ok := findNumber(parsed, "download"); ok {
				stats.DownloadBytes = v
			}
			// общие альтернативы
			if stats.UploadBytes == 0 {
				if v, ok := findNumber(parsed, "sent"); ok {
					stats.UploadBytes = v
				}
			}
			if stats.DownloadBytes == 0 {
				if v, ok := findNumber(parsed, "recv"); ok {
					stats.DownloadBytes = v
				}
			}
			if v, ok := findNumber(parsed, "ping"); ok {
				stats.Ping = int(v)
			}
			if v, ok := findNumber(parsed, "connection"); ok {
				stats.ConnectionCount = int(v)
			}
			stats.LastUpdate = time.Now()
			// Если нашли хотя бы что-то — вернём
			if stats.UploadBytes != 0 || stats.DownloadBytes != 0 || stats.Ping != 0 || stats.ConnectionCount != 0 {
				return stats, nil
			}
		}
	}

	// Фоллбек: попробуем спарсить access.log (если есть) — best-effort
	accessPath := x.getLogPath("access.log")
	f, err := os.Open(accessPath)
	if err != nil {
		return nil, fmt.Errorf("no stats available: %v", err)
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	var totalBytes int64
	var ips = make(map[string]struct{})
	for scanner.Scan() {
		line := scanner.Text()
		// ищем все IP-адреса в строке
		for _, part := range strings.Fields(line) {
			if ip := net.ParseIP(strings.Trim(part, ",[]()")); ip != nil {
				ips[ip.String()] = struct{}{}
			}
		}
		// ищем числа в конце строки как пример размера
		// naive: берем последнее число в строке
		parts := strings.Fields(line)
		if len(parts) > 0 {
			last := parts[len(parts)-1]
			if n, err := strconv.ParseInt(strings.Trim(last, ",."), 10, 64); err == nil {
				totalBytes += n
			}
		}
	}

	stats := &XrayStats{
		UploadBytes:     0,
		DownloadBytes:   totalBytes,
		Ping:            0,
		ConnectionCount: len(ips),
	}

	// попробуем измерить пинг быстрым TCP dial
	start := time.Now()
	conn, err := net.DialTimeout("tcp", "8.8.8.8:53", 2*time.Second)
	if err == nil {
		stats.Ping = int(time.Since(start).Milliseconds())
		conn.Close()
	}

	return stats, nil
}

// healthCheck проверяет работоспособность VPN
func (x *XrayManager) healthCheck() {
	time.Sleep(3 * time.Second)

	for x.isRunning {
		isHealthy := x.checkProxyHealth()
		if isHealthy {
			fmt.Println("✅ Proxy health check passed")
		} else {
			fmt.Println("⚠️ Proxy health check failed")
		}
		time.Sleep(30 * time.Second)
	}
}

// checkProxyHealth проверяет что прокси работает
func (x *XrayManager) checkProxyHealth() bool {
	conn, err := net.DialTimeout("tcp", "127.0.0.1:1080", 3*time.Second)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}

// Stop останавливает Xray
func (x *XrayManager) Stop() error {
	if !x.isRunning || x.process == nil {
		return nil
	}

	fmt.Println("🛑 Stopping Xray...")

	// Отключаем автоперезапуск
	x.restartCount = 10

	// Очищаем системный прокси
	x.ClearSystemProxy()

	// Graceful shutdown
	if err := x.process.Signal(os.Interrupt); err != nil {
		fmt.Printf("⚠️ Graceful shutdown failed, killing process: %v\n", err)
		x.process.Kill()
	}

	x.isRunning = false
	x.process = nil

	x.statsMu.Lock()
	x.stats = &XrayStats{}
	x.statsMu.Unlock()

	fmt.Println("✅ Xray stopped")
	return nil
}

// GetStats возвращает текущую статистику
func (x *XrayManager) GetStats() *XrayStats {
	x.statsMu.RLock()
	defer x.statsMu.RUnlock()
	if x.stats == nil {
		return &XrayStats{}
	}
	// return a copy to avoid races
	copy := *x.stats
	return &copy
}

// IsRunning возвращает статус работы
func (x *XrayManager) IsRunning() bool {
	return x.isRunning
}

// GetStatus возвращает детальный статус
func (x *XrayManager) GetStatus() string {
	if !x.isRunning {
		return "stopped"
	}

	if x.checkProxyHealth() {
		return "connected"
	}

	return "connecting"
}

func (x *XrayManager) getLogPath(filename string) string {
	logDir := filepath.Join(x.getXrayDir(), "logs")
	os.MkdirAll(logDir, 0755)
	return filepath.Join(logDir, filename)
}
