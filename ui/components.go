package ui

import (
	"fmt"
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
	connectButton   *widget.Button
	emptyLabel      *widget.Label
	themeSwitch     *widget.Button
	connections     []models.Connection
	selectedIndex   int
	isConnected     bool
	isDarkTheme     bool
	onUpdate        func()
	onThemeChange   func(bool) // Callback для смены темы
}

func NewComponents() *Components {
	comp := &Components{
		isConnected:   false,
		selectedIndex: -1,
		connections:   []models.Connection{},
		isDarkTheme:   true, // По умолчанию светлая тема
	}

	comp.loadConnections()
	comp.createComponents()
	return comp
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

	c.connectButton = widget.NewButton("Подключиться", c.toggleConnection)
	c.connectButton.Importance = widget.HighImportance

	c.emptyLabel = widget.NewLabel("Нет сохраненных подключений\n\nДобавьте первое подключение через меню \"Файл\"")
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
		c.connectButton.SetText("Подключиться")
		c.statusLabel.SetText("Готов к подключению")
		c.connectButton.Importance = widget.HighImportance
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
	if c.selectedIndex < 0 || c.selectedIndex >= len(c.connections) {
		return
	}

	c.isConnected = !c.isConnected
	if c.isConnected {
		c.connectButton.SetText("Отключиться")
		c.statusLabel.SetText("Подключено к " + c.connections[c.selectedIndex].Name)
		c.connectButton.Importance = widget.DangerImportance

		for i := range c.connections {
			if i == c.selectedIndex {
				c.connections[i].Status = "connected"
			} else {
				c.connections[i].Status = "disconnected"
			}
		}
	} else {
		c.connectButton.SetText("Подключиться")
		c.statusLabel.SetText("Отключено")
		c.connectButton.Importance = widget.HighImportance

		for i := range c.connections {
			c.connections[i].Status = "disconnected"
		}
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
