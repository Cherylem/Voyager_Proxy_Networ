package core

import (
	"bufio"
	"bytes"
	"context"
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
	"strconv"
	"strings"
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
	socksPort     int
	httpPort      int
	// stats control
	statsEnabled bool
	statsCtx     context.Context
	statsCancel  context.CancelFunc
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
		stats:        &XrayStats{},
		socksPort:    1080,
		httpPort:     1081,
		statsEnabled: false,
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

	// Reset preferred ports to standard defaults on each Start attempt
	x.socksPort = 1080
	x.httpPort = 1081

	// If standard ports are occupied, pick free ports dynamically.
	// We intentionally try standard ports first every Start as requested.
	if x.checkPortListening(x.socksPort) {
		newPort, err := getFreePort()
		if err == nil {
			fmt.Printf("⚠️ socks port %d busy, switching to %d\n", x.socksPort, newPort)
			x.socksPort = newPort
		} else {
			fmt.Printf("⚠️ socks port %d busy and failed to find free port: %v\n", x.socksPort, err)
		}
	}
	if x.checkPortListening(x.httpPort) {
		newPort, err := getFreePort()
		if err == nil {
			fmt.Printf("⚠️ http port %d busy, switching to %d\n", x.httpPort, newPort)
			x.httpPort = newPort
		} else {
			fmt.Printf("⚠️ http port %d busy and failed to find free port: %v\n", x.httpPort, err)
		}
	}

	// 2. Генерируем конфиг (uses x.socksPort/x.httpPort)
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

	// Start the process
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

	// 9. Настраиваем системный прокси ТОЛЬКО после проверки здоровья
	time.Sleep(1 * time.Second)
	if x.checkProxyHealth() {
		if err := x.SetupSystemProxy(); err != nil {
			fmt.Printf("⚠️ Failed to setup system proxy: %v\n", err)
		} else {
			fmt.Println("✅ System proxy configured")
		}
	} else {
		fmt.Println("⚠️ Xray proxy not responding, skipping system proxy setup")
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

	// 11. Запускаем мониторинг статистики только если включено
	if x.statsEnabled {
		// create a cancellable context for stats loop
		if x.statsCtx == nil {
			x.statsCtx, x.statsCancel = context.WithCancel(context.Background())
		}
		go x.monitorStats(x.statsCtx)
	}

	// health check always runs
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
	// Валидация: проверяем что Xray запущен
	if !x.checkProxyHealth() {
		return fmt.Errorf("xray proxy is not healthy, refusing to setup system proxy")
	}

	// Настройка HTTP прокси для Windows (только localhost)
	cmd := exec.Command("netsh", "winhttp", "set", "proxy", fmt.Sprintf("127.0.0.1:%d", x.httpPort))
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to set windows proxy: %v, output: %s", err, output)
	}
	fmt.Println("✅ Windows system proxy configured (localhost only)")
	return nil
}

func (x *XrayManager) setupMacOSProxy() error {
	// Валидация: проверяем что Xray действительно запущен и слушает на портах
	if !x.checkProxyHealth() {
		return fmt.Errorf("xray proxy is not healthy, refusing to setup system proxy")
	}

	// Получаем список сетевых сервисов
	networks, err := exec.Command("networksetup", "-listallnetworkservices").Output()
	if err != nil {
		return err
	}

	scanner := bufio.NewScanner(bytes.NewReader(networks))
	for scanner.Scan() {
		service := strings.TrimSpace(scanner.Text())
		if service != "" && !strings.Contains(service, "*") {
			// Настройка HTTP прокси (только localhost для безопасности)
			exec.Command("networksetup", "-setwebproxy", service, "127.0.0.1", fmt.Sprintf("%d", x.httpPort)).Run()
			// Настройка HTTPS прокси (только localhost)
			exec.Command("networksetup", "-setsecurewebproxy", service, "127.0.0.1", fmt.Sprintf("%d", x.httpPort)).Run()
			// Настройка SOCKS прокси (только localhost)
			exec.Command("networksetup", "-setsocksfirewallproxy", service, "127.0.0.1", fmt.Sprintf("%d", x.socksPort)).Run()

			// Включаем прокси
			exec.Command("networksetup", "-setwebproxystate", service, "on").Run()
			exec.Command("networksetup", "-setsecurewebproxystate", service, "on").Run()
			exec.Command("networksetup", "-setsocksfirewallproxystate", service, "on").Run()
		}
	}

	if err := scanner.Err(); err != nil {
		return err
	}

	fmt.Println("✅ macOS system proxy configured (localhost only)")
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

// TestConnection проверяет что прокси работает через защищенные каналы
func (x *XrayManager) TestConnection() bool {
	proxyURL, err := url.Parse(fmt.Sprintf("http://127.0.0.1:%d", x.httpPort))
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

	// Используем только защищенные HTTPS сервисы для проверки
	testURLs := []string{
		"https://api.ipify.org?format=json",
		"https://ifconfig.me/ip",
		"https://icanhazip.com",
	}

	for _, testURL := range testURLs {
		resp, err := client.Get(testURL)
		if err != nil {
			fmt.Printf("⚠️ Proxy test connection issue with %s: %v\n", testURL, err)
			continue
		}
		defer resp.Body.Close()

		if resp.StatusCode == 200 {
			body, _ := io.ReadAll(resp.Body)
			fmt.Printf("✅ Proxy test successful (%s): %s\n", testURL, string(body))
			return true
		}
	}

	fmt.Println("⚠️ All proxy tests failed, but connection may still work")
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
	ports := []int{x.socksPort, x.httpPort}
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

// getPortOwner attempts to find which process is listening on the given port (macOS/Linux via lsof).
func (x *XrayManager) getPortOwner(port int) string {
	// Only attempt on unix-like systems where lsof is available
	if runtime.GOOS == "windows" {
		return "port busy"
	}

	cmd := exec.Command("lsof", "-nP", "-iTCP:"+strconv.Itoa(port), "-sTCP:LISTEN")
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	if err := cmd.Run(); err != nil {
		return strings.TrimSpace(out.String())
	}
	return strings.TrimSpace(out.String())
}

// getFreePort asks the kernel for an available port by binding to :0 and returning the assigned port.
func getFreePort() (int, error) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}
	defer l.Close()
	addr := l.Addr().(*net.TCPAddr)
	return addr.Port, nil
}

// monitorStats собирает статистику из Xray API
func (x *XrayManager) monitorStats(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			fmt.Println("[monitorStats] stopped by context")
			return
		case <-ticker.C:
			if !x.isRunning {
				continue
			}
			stats, err := x.getStats()
			if err == nil && stats != nil {
				x.statsMu.Lock()
				x.stats = stats
				if x.stats.LastUpdate.IsZero() {
					x.stats.LastUpdate = time.Now()
				}
				x.statsMu.Unlock()
				// Debug: print received snapshot
				fmt.Printf("[monitorStats] snapshot: upload=%d download=%d ping=%d connections=%d\n",
					stats.UploadBytes, stats.DownloadBytes, stats.Ping, stats.ConnectionCount)
			}
			if err != nil {
				fmt.Printf("[monitorStats] getStats error: %v\n", err)
			}
		}
	}
}

// getStats получает статистику через Xray API (только с localhost)
func (x *XrayManager) getStats() (*XrayStats, error) {
	// Попытка получить статистику только с localhost для безопасности
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

		// Debug: print HTTP body (truncated)
		bodyStr := string(body)
		if len(bodyStr) > 2000 {
			bodyStr = bodyStr[:2000] + "...(truncated)"
		}
		fmt.Printf("[getStats] HTTP %s -> %s\n", u, bodyStr)

		// Пытаемся распарсить JSON
		var parsed interface{}
		if err := json.Unmarshal(body, &parsed); err == nil {
			stats := &XrayStats{}
			var foundUpload, foundDownload, foundPing, foundConnections bool
			var walk func(interface{})
			walk = func(v interface{}) {
				switch t := v.(type) {
				case map[string]interface{}:
					for k, val := range t {
						lk := strings.ToLower(k)
						// try extract numeric value
						switch nv := val.(type) {
						case float64:
							if !foundUpload && (strings.Contains(lk, "upload") || strings.Contains(lk, "uplink") || strings.Contains(lk, "sent")) {
								stats.UploadBytes = int64(nv)
								foundUpload = true
							}
							if !foundDownload && (strings.Contains(lk, "download") || strings.Contains(lk, "downlink") || strings.Contains(lk, "recv") || strings.Contains(lk, "received")) {
								stats.DownloadBytes = int64(nv)
								foundDownload = true
							}
							if !foundPing && strings.Contains(lk, "ping") {
								stats.Ping = int(nv)
								foundPing = true
							}
							if !foundConnections && (strings.Contains(lk, "connection") || strings.Contains(lk, "connections")) {
								stats.ConnectionCount = int(nv)
								foundConnections = true
							}
						case string:
							if n, err := strconv.ParseInt(nv, 10, 64); err == nil {
								if !foundUpload && (strings.Contains(lk, "upload") || strings.Contains(lk, "uplink") || strings.Contains(lk, "sent")) {
									stats.UploadBytes = n
									foundUpload = true
								}
								if !foundDownload && (strings.Contains(lk, "download") || strings.Contains(lk, "downlink") || strings.Contains(lk, "recv") || strings.Contains(lk, "received")) {
									stats.DownloadBytes = n
									foundDownload = true
								}
							}
						}
						walk(val)
					}
				case []interface{}:
					for _, it := range t {
						walk(it)
					}
				}
			}
			walk(parsed)

			if foundUpload || foundDownload || foundPing || foundConnections {
				stats.LastUpdate = time.Now()
				fmt.Printf("[getStats] parsed stats: upload=%d download=%d ping=%d connections=%d\n", stats.UploadBytes, stats.DownloadBytes, stats.Ping, stats.ConnectionCount)
				return stats, nil
			}
		}
	}

	// Фоллбек: попробуем спарсить access.log (если есть)
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
		// ищем числа в конце строки
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
	fmt.Printf("[getStats] fallback to access.log: totalBytes=%d connections=%d\n", totalBytes, len(ips))

	// Измеряем пинг через DNS запрос (защищенный)
	start := time.Now()
	resolver := &net.Resolver{
		PreferGo: true,
	}
	_, err = resolver.LookupHost(context.Background(), "google.com")
	if err == nil {
		stats.Ping = int(time.Since(start).Milliseconds())
	}

	return stats, nil
}

// EnableStats turns on or off the internal stats polling. When enabled and Xray is running,
// a background goroutine will poll stats. When disabled, polling is stopped to save resources.
func (x *XrayManager) EnableStats(enabled bool) {
	if enabled == x.statsEnabled {
		return
	}
	x.statsEnabled = enabled
	if enabled {
		// create context and start polling if process already running
		if x.statsCtx == nil {
			x.statsCtx, x.statsCancel = context.WithCancel(context.Background())
		}
		if x.isRunning {
			go x.monitorStats(x.statsCtx)
		}
	} else {
		if x.statsCancel != nil {
			x.statsCancel()
			x.statsCancel = nil
			x.statsCtx = nil
		}
	}
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
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%d", x.socksPort), 3*time.Second)
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

	if err := x.process.Signal(os.Interrupt); err != nil {
		fmt.Printf("⚠️ Graceful shutdown failed, killing process: %v\n", err)
		x.process.Kill()
	}

	if x.statsCancel != nil {
		x.statsCancel()
		x.statsCancel = nil
		x.statsCtx = nil
	}

	x.isRunning = false
	x.process = nil

	x.statsMu.Lock()
	x.stats = &XrayStats{}
	x.statsMu.Unlock()

	fmt.Println("✅ Xray stopped")
	return nil
}

func (x *XrayManager) GetStats() *XrayStats {
	x.statsMu.RLock()
	defer x.statsMu.RUnlock()
	if x.stats == nil {
		return &XrayStats{}
	}
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
