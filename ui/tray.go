package ui

import (
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/driver/desktop"
)

type TrayManager struct {
	app        fyne.App
	window     fyne.Window
	menu       *fyne.Menu
	components *Components
	vpnApp     *VPNApp // Добавляем ссылку на главное приложение
}

func NewTrayManager(app fyne.App, window fyne.Window, components *Components, vpnApp *VPNApp) *TrayManager {
	tm := &TrayManager{
		app:        app,
		window:     window,
		components: components,
		vpnApp:     vpnApp, // Добавляем ссылку на главное приложение
	}
	tm.createTrayMenu()
	return tm
}

// Добавляем метод для установки ссылки на VPNApp
func (tm *TrayManager) SetVPNApp(vpnApp *VPNApp) {
	tm.vpnApp = vpnApp
}

func (tm *TrayManager) showWindow() {
	if tm.vpnApp != nil {
		tm.vpnApp.showWindow() // Используем метод из VPNApp
	} else {
		tm.window.Show() // Фолбэк
	}
}

func (tm *TrayManager) toggleConnection() {
	if tm.components.IsConnected() {
		tm.components.Disconnect()
	} else {
		tm.components.Connect()
	}
	tm.UpdateMenu()
}

func (tm *TrayManager) quitApp() {
	tm.app.Quit()
}

func (tm *TrayManager) createTrayMenu() {
	tm.menu = fyne.NewMenu("VPN Client")
	tm.UpdateMenu()

	if desk, ok := tm.app.(desktop.App); ok {
		// Устанавливаем иконку трея
		tm.setTrayIcon(false) // начальное состояние - отключено

		desk.SetSystemTrayMenu(tm.menu)
	}
}

// Метод для установки иконки трея
func (tm *TrayManager) setTrayIcon(connected bool) {
	if desk, ok := tm.app.(desktop.App); ok {
		var iconPath string
		if connected {
			iconPath = "assets/white_icon.png"
		} else {
			iconPath = "assets/tray_purple_icon.png"
		}

		icon, err := fyne.LoadResourceFromPath(iconPath)
		if err != nil {
			fmt.Printf("Не удалось загрузить иконку трея: %v\n", err)
			return
		}

		desk.SetSystemTrayIcon(icon)
		fmt.Printf("Иконка трея установлена: %s\n", iconPath)
	}
}

func (tm *TrayManager) UpdateMenu() {
	connectText := "Подключиться"
	isConnected := tm.components.IsConnected()

	if isConnected {
		connectText = "Отключиться"
	}

	connectItem := fyne.NewMenuItem(connectText, tm.toggleConnection)
	showItem := fyne.NewMenuItem("Показать окно", tm.showWindow)
	quitItem := fyne.NewMenuItem("Выход", tm.quitApp)

	tm.menu.Items = []*fyne.MenuItem{
		connectItem,
		fyne.NewMenuItemSeparator(),
		showItem,
		quitItem,
	}

	// Обновляем иконку трея при изменении статуса
	tm.setTrayIcon(isConnected)

	if desk, ok := tm.app.(desktop.App); ok {
		desk.SetSystemTrayMenu(tm.menu)
	}
}
