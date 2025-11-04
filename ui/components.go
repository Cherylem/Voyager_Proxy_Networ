package ui

import (
	"fmt"
	"os"
	"strings"
	"time"
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

	// Создаем кликабельное изображение-кнопку (даже если иконки нет - создаем пустой)
	c.connectButton = NewImageButton(startIcon, c.toggleConnection)
	// Поставим удобный размер по умолчанию — можно изменить позднее
	c.connectButton.SetSize(fyne.NewSize(96, 96))
	// Устанавливаем корректную иконку состояния (по умолчанию отключено)
	c.updateConnectIcon(false)

	// Чекбокс для включения/выключения отображения статистики
	c.showStats = false
	c.showStatsCheck = widget.NewCheck("Показывать статистику(в разработке)", func(checked bool) {
		c.showStats = checked
		// tell vpn manager to enable/disable stats collection to save resources
		if c.vpnManager != nil {
			c.vpnManager.EnableStats(checked)
		}
		if !checked {
			// очистим предыдущие значения и UI
			c.prevStats = nil
			fyne.Do(func() {
				c.statusLabel.SetText("Статистика отключена")
			})
		}
	})
	c.showStatsCheck.SetChecked(false)

	c.emptyLabel = widget.NewLabel("Нет подключений\nДобавьте через меню \"Файл\"")
	c.emptyLabel.Alignment = fyne.TextAlignCenter
	c.emptyLabel.Wrapping = fyne.TextWrapWord

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
			// если нет стартовой иконки, очистим изображение
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
		// nothing to change for ImageButton importance
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
		// Обновим UI немедленно после остановки
		c.updateStats()
	}
	c.connectionsList.Refresh()
	if c.onUpdate != nil {
		c.onUpdate() // это уже должно обновлять трей
	}
}

func (c *Components) GetMainContent() fyne.CanvasObject {
	title := widget.NewLabel("VPN Client")
	title.TextStyle = fyne.TextStyle{Bold: true}
	title.Alignment = fyne.TextAlignCenter

	// Верхняя панель с заголовком и переключателем темы
	header := container.NewHBox(
		layout.NewSpacer(),
		title,
		layout.NewSpacer(),
		c.themeSwitch, // Переключатель темы в правом углу
	)

	statusCard := container.NewVBox(
		container.NewCenter(c.statusLabel),
		container.NewCenter(c.connectButton),
		container.NewCenter(c.showStatsCheck),
	)

	connectionsTitle := widget.NewLabel("Доступные подключения")
	connectionsTitle.TextStyle = fyne.TextStyle{Bold: true}

	var listContent fyne.CanvasObject
	if len(c.connections) == 0 {
		listContent = container.NewCenter(c.emptyLabel)
	} else {
		listContent = container.NewStack(
			container.NewPadded(c.connectionsList),
		)
	}

	mainContent := container.NewBorder(
		container.NewVBox(
			container.NewPadded(header), // Новый header с переключателем
			layout.NewSpacer(),
			container.NewPadded(statusCard),
			layout.NewSpacer(),
			widget.NewSeparator(),
			container.NewPadded(connectionsTitle),
			widget.NewSeparator(),
		),
		nil,
		nil,
		nil,
		listContent,
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
