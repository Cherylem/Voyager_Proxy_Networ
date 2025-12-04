package ui

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"strings"
	"time"
	"golang.org/x/net/publicsuffix"
	"vpn-client/core"
	"vpn-client/models"
	"vpn-client/services"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
)

type Components struct {
	connectionsList *widget.List
	statusLabel     *widget.Label
	connectButton   *ImageButton
	emptyLabel      *widget.Label
	themeSwitch     *widget.Button
	showStatsCheck  *widget.Check
	connections     []models.Connection
	selectedIndex   int
	isConnected     bool
	isDarkTheme     bool
	onUpdate        func()
	onThemeChange   func(bool) // Callback для смены темы
	vpnManager      *core.XrayManager
	statsTimer      *time.Timer
	showStats       bool
	prevStats       *core.XrayStats
}

func NewComponents() *Components {
	comp := &Components{
		isConnected:   false,
		selectedIndex: -1,
		connections:   []models.Connection{},
		isDarkTheme:   true, // По умолчанию светлая тема
		vpnManager:    core.NewXrayManager(),
	}

	comp.loadConnections()
	comp.createComponents()
	return comp
}

func (c *Components) startStatsUpdate() {
	c.statsTimer = time.AfterFunc(1*time.Second, func() {
		c.updateStats()
		c.startStatsUpdate() // Перезапускаем таймер
	})
}

func (c *Components) updateStats() {
	// Всегда обновляем UI — переводим статус на русский и показываем/скрываем статистику
	stats := c.vpnManager.GetStats()
	status := c.vpnManager.GetStatus()

	// Перевод статуса на русский
	var statusRus string
	switch strings.ToLower(status) {
	case "connected":
		statusRus = "Подключено"
	case "connecting":
		statusRus = "Подключение"
	case "stopped", "disconnected", "":
		statusRus = "Отключено"
	default:
		statusRus = status
	}

	// Если не запущено — показываем только статус Отключено и очищаем prevStats
	if !c.vpnManager.IsRunning() {
		c.prevStats = nil
		fyne.Do(func() {
			c.statusLabel.SetText(statusRus)
		})
		return
	}

	// Если пользователь выключил отображение статистики — показываем соответствующий текст
	if !c.showStats {
		fyne.Do(func() {
			c.statusLabel.SetText(statusRus + "\nСтатистика отключена")
		})
		return
	}

	// Получаем новые накопительные значения от менеджера
	cur := stats

	// Если нет предыдущих данных — сохранем и покажем базовый статус
	if c.prevStats == nil {
		c.prevStats = cur
		fyne.Do(func() {
			c.statusLabel.SetText(statusRus + "\nСбор статистики...")
		})
		return
	}

	// Рассчитываем скорость как дельта байт / дельта секунд
	deltaSec := cur.LastUpdate.Sub(c.prevStats.LastUpdate).Seconds()
	if deltaSec <= 0 {
		deltaSec = 1
	}

	uploadRate := int64(0)
	downloadRate := int64(0)
	if cur.UploadBytes >= c.prevStats.UploadBytes {
		uploadRate = int64(float64(cur.UploadBytes-c.prevStats.UploadBytes) / deltaSec)
	}
	if cur.DownloadBytes >= c.prevStats.DownloadBytes {
		downloadRate = int64(float64(cur.DownloadBytes-c.prevStats.DownloadBytes) / deltaSec)
	}

	statusText := fmt.Sprintf("%s\n⬆️ %s/s ⬇️ %s/s\n📡 Пинг: %dms",
		statusRus, formatBytes(uploadRate), formatBytes(downloadRate), cur.Ping)

	// Сохраняем текущие данные как предыдущие для следующего расчёта
	c.prevStats = cur

	// Обновляем UI
	fyne.Do(func() {
		c.statusLabel.SetText(statusText)
	})
}

func formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

func (c *Components) SetOnUpdateCallback(callback func()) {
	c.onUpdate = callback
}

func (c *Components) SetOnThemeChangeCallback(callback func(bool)) {
	c.onThemeChange = callback
}

func (c *Components) loadConnections() {
	connections, err := services.GetConnections()
	if err == nil {
		c.connections = connections
		if len(c.connections) > 0 {
			c.selectedIndex = 0
		}
	}
}

func (c *Components) createComponents() {
	c.statusLabel = widget.NewLabel("Готов к подключению")
	c.statusLabel.Alignment = fyne.TextAlignCenter
	wd, _ := os.Getwd()
	fmt.Println("Current working directory:", wd)
	// Попытка загрузить иконку для кнопки подключения
	connectIcon, err1 := fyne.LoadResourceFromPath("assets/white_icon.png")

	var startIcon fyne.Resource
	if err1 == nil {
		startIcon = connectIcon
	}

	// Создаем кликабельное изображение-кнопку
	c.connectButton = NewImageButton(startIcon, c.toggleConnection)
	c.connectButton.SetSize(fyne.NewSize(96, 96))
	c.updateConnectIcon(false)

	// Чекбокс для включения/выключения отображения статистики
	c.showStats = false
	c.showStatsCheck = widget.NewCheck("Показывать статистику(в разработке)", func(checked bool) {
		c.showStats = checked
		if c.vpnManager != nil {
			c.vpnManager.EnableStats(checked)
		}
		if !checked {
			c.prevStats = nil
			fyne.Do(func() {
				c.statusLabel.SetText("Статистика отключена")
			})
		}
	})
	c.showStatsCheck.SetChecked(false)

	c.emptyLabel = widget.NewLabelWithStyle(
		"Нет подключений\nДобавьте через меню \"Файл\"",
		fyne.TextAlignCenter,        // выравнивание по центру
		fyne.TextStyle{Bold: false}, // стиль
	)
	// Переключатель темы
	c.themeSwitch = widget.NewButton("🌙", c.toggleTheme)
	c.themeSwitch.Resize(fyne.NewSize(50, 80))
	c.updateThemeButton()

	c.connectionsList = widget.NewList(
		func() int { return len(c.connections) },
		func() fyne.CanvasObject {
			return container.NewHBox(
				widget.NewCheck("", nil),
				container.NewVBox(
					widget.NewLabel(""),
					widget.NewLabel(""),
				),
				layout.NewSpacer(),
				widget.NewButton("🗑️", nil),
			)
		},
		func(i widget.ListItemID, o fyne.CanvasObject) {
			cont := o.(*fyne.Container)
			conn := c.connections[i]

			check := cont.Objects[0].(*widget.Check)
			textContainer := cont.Objects[1].(*fyne.Container)
			nameLabel := textContainer.Objects[0].(*widget.Label)
			detailsLabel := textContainer.Objects[1].(*widget.Label)
			deleteButton := cont.Objects[3].(*widget.Button)

			check.SetChecked(i == c.selectedIndex)
			check.OnChanged = func(checked bool) {
				if checked {
					c.selectedIndex = i
					c.connectionsList.Refresh()
				}
			}

			nameLabel.SetText(conn.Name)
			nameLabel.TextStyle = fyne.TextStyle{Bold: true}

			details := fmt.Sprintf("%s:%d • %s", conn.Country, conn.Port, conn.Security)
			if conn.SNI != "" {
				details += " • " + conn.SNI
			}
			detailsLabel.SetText(details)
			detailsLabel.TextStyle = fyne.TextStyle{Italic: true}

			deleteButton.OnTapped = func() {
				c.deleteConnection(i)
			}
		},
	)
}

func (c *Components) toggleTheme() {
	c.isDarkTheme = !c.isDarkTheme
	c.updateThemeButton()

	if c.onThemeChange != nil {
		c.onThemeChange(c.isDarkTheme)
	}
}

func (c *Components) updateThemeButton() {
	if c.isDarkTheme {
		c.themeSwitch.SetText("☀️") // Солнышко для светлой темы
	} else {
		c.themeSwitch.SetText("🌙") // Луна для темной темы
	}
}

// Обновляет иконку кнопки подключения в зависимости от состояния
func (c *Components) updateConnectIcon(connected bool) {
	if c.connectButton == nil {
		return
	}
	// Попытка загрузить иконки
	connectIcon, _ := fyne.LoadResourceFromPath("assets/tray_purple_icon.png")
	disconnectIcon, _ := fyne.LoadResourceFromPath("assets/white_icon.png")

	// Если ни одной иконки нет — просто ничего не делаем
	if connectIcon == nil && disconnectIcon == nil {
		return
	}

	if connected {
		if disconnectIcon != nil {
			c.connectButton.SetResource(disconnectIcon)
		}
	} else {
		if connectIcon != nil {
			c.connectButton.SetResource(connectIcon)
		} else {
			c.connectButton.SetResource(nil)
		}
	}
}

func (c *Components) deleteConnection(index int) {
	if index < 0 || index >= len(c.connections) {
		return
	}

	connectionName := c.connections[index].Name
	c.connections = append(c.connections[:index], c.connections[index+1:]...)

	if c.selectedIndex == index {
		c.selectedIndex = -1
	} else if c.selectedIndex > index {
		c.selectedIndex--
	}

	if len(c.connections) == 0 {
		c.isConnected = false
		c.updateConnectIcon(false)
		c.statusLabel.SetText("Готов к подключению")
	}

	config := &services.AppConfig{Connections: c.connections}
	err := services.SaveConfig(config)
	if err != nil {
		fmt.Printf("Ошибка сохранения: %v\n", err)
		return
	}

	c.connectionsList.Refresh()

	if c.onUpdate != nil {
		c.onUpdate()
	}

	fmt.Printf("Удалено подключение: %s\n", connectionName)
}

func (c *Components) toggleConnection() {
	if c.selectedIndex < 0 {
		return
	}
	if !c.vpnManager.IsRunning() {
		// Подключение
		conn := c.connections[c.selectedIndex]
		err := c.vpnManager.Start(&conn)
		if err != nil {
			fmt.Printf("❌ Connection failed: %v\n", err)
			c.statusLabel.SetText("Ошибка подключения")
			return
		}
		c.updateConnectIcon(true)
		c.isConnected = true
		c.startStatsUpdate() // Запускаем обновление статистики
	} else {
		// Отключение
		err := c.vpnManager.Stop()
		if err != nil {
			fmt.Printf("❌ Disconnection failed: %v\n", err)
			return
		}
		c.updateConnectIcon(false)
		c.isConnected = false
		if c.statsTimer != nil {
			c.statsTimer.Stop()
		}
		c.updateStats()
	}
	c.connectionsList.Refresh()
	if c.onUpdate != nil {
		c.onUpdate()
	}
}

func (c *Components) GetMainContent() fyne.CanvasObject {
	title := widget.NewLabel("VPN Client")
	title.TextStyle = fyne.TextStyle{Bold: true}
	title.Alignment = fyne.TextAlignCenter

	// Кнопка для управления whitelist
	whitelistBtn := widget.NewButton("✅ Whitelist", func() {
		c.showWhitelistDialog()
	})

	// Верхняя панель: кнопка слева, заголовок отцентрирован, переключатель темы справа
	header := container.NewBorder(
		nil, nil,
		whitelistBtn,
		c.themeSwitch,
		container.NewCenter(title),
	)

	statusCard := container.NewVBox(
		container.NewCenter(c.statusLabel),
		container.NewCenter(c.connectButton),
		container.NewCenter(c.showStatsCheck),
	)

	connectionsTitle := widget.NewLabel("Доступные подключения")
	connectionsTitle.TextStyle = fyne.TextStyle{Bold: true}

	//Когда подключений нет - не показываем список вообще, показываем только сообщение
	var content fyne.CanvasObject
	if len(c.connections) == 0 {
		content = container.NewCenter(c.emptyLabel)
	} else {
		content = container.NewBorder(
			container.NewVBox(
				container.NewPadded(connectionsTitle),
				widget.NewSeparator(),
			),
			nil, nil, nil,
			container.NewStack(container.NewPadded(c.connectionsList)),
		)
	}

	mainContent := container.NewBorder(
		container.NewVBox(
			container.NewPadded(header),
			layout.NewSpacer(),
			container.NewPadded(statusCard),
			layout.NewSpacer(),
			widget.NewSeparator(),
		),
		nil,
		nil,
		nil,
		content,
	)

	return container.NewPadded(mainContent)
}

func (c *Components) IsConnected() bool {
	return c.isConnected
}

func (c *Components) Connect() {
	if !c.isConnected && c.selectedIndex >= 0 && c.selectedIndex < len(c.connections) {
		c.toggleConnection()
	}
}

func (c *Components) Disconnect() {
	if c.isConnected {
		c.toggleConnection()
	}
}

func (c *Components) AddConnection(conn models.Connection) error {
	c.connections = append(c.connections, conn)

	if len(c.connections) == 1 {
		c.selectedIndex = 0
	}

	c.connectionsList.Refresh()

	config := &services.AppConfig{Connections: c.connections}
	err := services.SaveConfig(config)

	if c.onUpdate != nil {
		c.onUpdate()
	}

	return err
}

func (c *Components) GetConnections() []models.Connection {
	return c.connections
}

func (c *Components) SetConnections(connections []models.Connection) {
	c.connections = connections
	c.connectionsList.Refresh()

	if c.onUpdate != nil {
		c.onUpdate()
	}
}

func (c *Components) GetSelectedConnection() *models.Connection {
	if c.selectedIndex < 0 || c.selectedIndex >= len(c.connections) {
		return nil
	}
	return &c.connections[c.selectedIndex]
}

// showWhitelistDialog показывает окно для управления whitelist сайтов
func (c *Components) showWhitelistDialog() {
	app := fyne.CurrentApp()
	whitelistWindow := app.NewWindow("Whitelist сайтов")
	whitelistWindow.Resize(fyne.NewSize(600, 500))

	// Загружаем whitelist
	whitelist := c.loadWhitelist()

	// Переменная для ссылки на список (нужна для обновления в callback)
	var whitelistList *widget.List

	// Создаем список сайтов
	whitelistList = widget.NewList(
		func() int { return len(whitelist) },
		func() fyne.CanvasObject {
			return container.NewVBox(
				container.NewHBox(
					widget.NewLabel(""),
					layout.NewSpacer(),
					widget.NewButton("🗑️", nil),
				),
				widget.NewSeparator(),
			)
		},
		func(i widget.ListItemID, o fyne.CanvasObject) {
			cont := o.(*fyne.Container)
			itemContainer := cont.Objects[0].(*fyne.Container)
			urlLabel := itemContainer.Objects[0].(*widget.Label)
			deleteBtn := itemContainer.Objects[2].(*widget.Button)

			urlLabel.SetText(whitelist[i])

			deleteBtn.OnTapped = func() {
				whitelist = append(whitelist[:i], whitelist[i+1:]...)
				whitelistList.Refresh()
			}
		},
	)

	// Поле для добавления новой записи
	newUrlEntry := widget.NewEntry()
	newUrlEntry.SetPlaceHolder("https://example.com")

	// Кнопка добавления
	addBtn := widget.NewButton("➕ Добавить сайт", func() {
		if newUrlEntry.Text == "" {
			fmt.Println("❌ URL не может быть пустым")
			return
		}

		whitelist = append(whitelist, newUrlEntry.Text)
		whitelistList.Refresh()
		newUrlEntry.SetText("")
	})

	// Кнопки сохранения и отмены
	saveBtn := widget.NewButton("💾 Сохранить", func() {
		c.saveWhitelist(whitelist)
		fmt.Println("✅ Whitelist сохранен")
		whitelistWindow.Close()
	})

	cancelBtn := widget.NewButton("Отмена", func() {
		whitelistWindow.Close()
	})

	// Информационный текст
	infoLabel := widget.NewLabelWithStyle(
		"Добавьте сайты, которые будут открываться напрямую (без VPN).\nВсе остальные сайты будут использовать VPN.",
		fyne.TextAlignCenter,
		fyne.TextStyle{Italic: true},
	)

	buttons := container.NewHBox(
		layout.NewSpacer(),
		saveBtn,
		cancelBtn,
	)

	addSection := container.NewVBox(
		newUrlEntry,
		addBtn,
	)

	// Разметка: информационный текст сверху, кнопки снизу, центр — список сайтов с секцией добавления
	// Оборачиваем список в вертикальный скролл и задаем минимальный размер,
	// чтобы он занимал доступное пространство в окне (аналогично списку подключений).
	vscroll := container.NewVScroll(whitelistList)
	vscroll.SetMinSize(fyne.NewSize(560, 280))

	listSection := container.NewBorder(
		widget.NewLabel("Сайты в списке:"),
		nil, nil, nil,
		vscroll,
	)

	center := container.NewVBox(
		addSection,
		widget.NewSeparator(),
		listSection,
	)

	content := container.NewBorder(
		infoLabel,
		buttons,
		nil,
		nil,
		center,
	)

	// Оборачиваем в Padded для отступов и устанавливаем как контент окна
	whitelistWindow.SetContent(container.NewPadded(content))
	whitelistWindow.Show()
}

// loadWhitelist загружает список whitelist сайтов
func (c *Components) loadWhitelist() []string {
	configPath := "configurations/whitelist.json"
	data, err := os.ReadFile(configPath)
	if err != nil {
		return []string{}
	}

	var whitelist []string
	err = json.Unmarshal(data, &whitelist)
	if err != nil {
		return []string{}
	}

	return whitelist
}

// saveWhitelist сохраняет список whitelist сайтов
func (c *Components) saveWhitelist(whitelist []string) {
	expanded := expandWhitelistEntries(whitelist)
	data, _ := json.MarshalIndent(expanded, "", "  ")
	os.WriteFile("configurations/whitelist.json", data, 0644)
}

// expandWhitelistEntries принимает список пользовательских записей (хосты или URL)
// и возвращает развернутый, уникальный список вариантов для записи в JSON.
// Для каждого хоста добавляются варианты: `host`, `*.host`, `baseDomain`, `*.baseDomain`.
func expandWhitelistEntries(raw []string) []string {
	out := make([]string, 0, len(raw)*4)
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

		add := func(val string) {
			if val == "" {
				return
			}
			if !seen[val] {
				seen[val] = true
				out = append(out, val)
			}
		}

		// основной домен
		add(s)

		// wildcard для поддоменов
		add("*." + s)

		// базовый домен: используем publicsuffix.EffectiveTLDPlusOne чтобы получить
		// registrable domain (eTLD+1). Это предотвращает добавление одиночных TLD
		// (например, "ru") в качестве правила.
		if etld1, err := publicsuffix.EffectiveTLDPlusOne(s); err == nil {
			if etld1 != s {
				add(etld1)
				add("*." + etld1)
			}
		}
	}

	return out
}
