package ui

import (
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"

	"vpn-client/models"
	"vpn-client/services"
	"vpn-client/themes"
)

type VPNApp struct {
	app         fyne.App
	window      fyne.Window
	components  *Components
	tray        *TrayManager
	isDarkTheme bool
}

func NewVPNApp() *VPNApp {
	vpnApp := &VPNApp{
		isDarkTheme: true,
	}
	vpnApp.initialize()
	return vpnApp
}

func (v *VPNApp) initialize() {
	v.app = app.NewWithID("com.vpn.client")

	// создаем окно
	v.window = v.app.NewWindow("Voyager Proxy Network")
	v.window.SetMaster()
	v.window.Resize(fyne.NewSize(600, 700))
	v.window.CenterOnScreen()

	// устанавливаем иконку окна
	v.setWindowIcon()

	// применяем тему (после создания окна)
	v.applyTheme()

	// При закрытии крестиком - скрываем окно
	v.window.SetCloseIntercept(func() {
		fmt.Println("Close button clicked - hiding window")
		v.hideWindow()
	})

	v.components = NewComponents()
	v.components.SetOnUpdateCallback(func() {
		v.window.SetContent(v.components.GetMainContent())
		if v.tray != nil {
			v.tray.UpdateMenu()
		}
	})
	v.components.SetOnThemeChangeCallback(v.onThemeChanged)

	// создаем tray менеджер (передаем vpnApp)
	v.tray = NewTrayManager(v.app, v.window, v.components, v)

	v.createAppMenu()
	v.window.SetContent(v.components.GetMainContent())

	v.window.Show()
}

// Добавляем метод для установки иконки окна
func (v *VPNApp) setWindowIcon() {
	// Пробуем загрузить иконку из assets
	icon, err := fyne.LoadResourceFromPath("assets/white_icon.png")
	if err != nil {
		fmt.Printf("Не удалось загрузить иконку окна: %v\n", err)

		// Создаем простую иконку как fallback
		fallbackIcon := &fyne.StaticResource{
			StaticName:    "64X64.png",
			StaticContent: []byte{},
		}
		v.window.SetIcon(fallbackIcon)
		return
	}

	v.window.SetIcon(icon)
	fmt.Println("Иконка окна установлена успешно")
}

// Пересоздание окна если оно было уничтожено
func (v *VPNApp) recreateWindow() {
	fmt.Println("Recreating window...")

	v.window = v.app.NewWindow("VPN Client")
	v.window.SetMaster()
	v.window.Resize(fyne.NewSize(500, 650))
	v.window.CenterOnScreen()

	v.window.SetCloseIntercept(func() {
		fmt.Println("Close button clicked on recreated window - hiding")
		v.hideWindow()
	})

	v.window.SetContent(v.components.GetMainContent())
}

// Метод для показа окна при клике в Dock
func (v *VPNApp) showWindowFromDock() {
	fmt.Println("Dock click detected - showing window")
	v.showWindow()
}

func (v *VPNApp) applyTheme() {
	v.app.Settings().SetTheme(themes.NewMinimalTheme(v.isDarkTheme))
}

func (v *VPNApp) onThemeChanged(isDark bool) {
	v.isDarkTheme = isDark
	v.applyTheme()
	v.window.SetContent(v.components.GetMainContent())
}

// Скрываем окно вместо закрытия
func (v *VPNApp) hideWindow() {
	v.window.Hide()
}

func (v *VPNApp) createAppMenu() {
	fileMenu := fyne.NewMenu("Файл",
		fyne.NewMenuItem("Новое подключение", v.showAddConnectionDialog),
		fyne.NewMenuItemSeparator(),
		fyne.NewMenuItem("Скрыть окно", v.hideWindow), // Явное скрытие
		fyne.NewMenuItem("Выход", v.quitApp),
	)

	connectionsMenu := fyne.NewMenu("Подключения",
		fyne.NewMenuItem("Копировать ссылку", v.copyConnectionLink),
		fyne.NewMenuItem("Импорт конфигурации", v.showAddConnectionDialog),
	)

	mainMenu := fyne.NewMainMenu(fileMenu, connectionsMenu)
	v.window.SetMainMenu(mainMenu)
}

// Метод для показа окна
func (v *VPNApp) showWindow() {
	v.window.Show()
	v.window.RequestFocus() // Фокусируем окно
}

func (v *VPNApp) quitApp() {
	v.app.Quit()
}

func (v *VPNApp) showAddConnectionDialog() {
	nameEntry := widget.NewEntry()
	nameEntry.SetPlaceHolder("Название подключения (опционально)")

	configEntry := widget.NewMultiLineEntry()
	configEntry.SetPlaceHolder("vless://uuid@server:port?security=reality&sni=domain.com#Название")
	configEntry.Wrapping = fyne.TextWrapWord

	statusLabel := widget.NewLabel("")
	statusLabel.Wrapping = fyne.TextWrapWord
	statusLabel.Alignment = fyne.TextAlignCenter

	// Заголовок
	title := widget.NewLabelWithStyle(
		"Новое подключение",
		fyne.TextAlignCenter,
		fyne.TextStyle{Bold: true},
	)

	// Секция информации
	infoLabel := widget.NewLabelWithStyle(
		"Введите VLESS ссылку для добавления нового подключения",
		fyne.TextAlignCenter,
		fyne.TextStyle{Italic: true},
	)

	// Поля ввода с подписями
	formContent := container.NewVBox(
		widget.NewLabel("Название (опционально):"),
		nameEntry,
		widget.NewLabel("VLESS ссылка:"),
		configEntry,
		widget.NewSeparator(),
	)

	// Статус сервера
	statusCard := container.NewVBox(
		container.NewCenter(statusLabel),
	)

	// Основной контент
	content := container.NewVBox(
		title,
		infoLabel,
		widget.NewSeparator(),
		formContent,
		statusCard,
	)

	// Кнопки
	cancelButton := widget.NewButton("Отмена", nil)
	addButton := widget.NewButton("Добавить", nil)

	// Стилизация кнопок (используем константы из пакета widget)
	cancelButton.Importance = widget.LowImportance
	addButton.Importance = widget.HighImportance

	buttons := container.NewHBox(
		layout.NewSpacer(),
		cancelButton,
		addButton,
	)

	dialogContent := container.NewBorder(nil, buttons, nil, nil, content)
	dialog := widget.NewModalPopUp(dialogContent, v.window.Canvas())

	dialog.Resize(fyne.NewSize(600, 550))

	validateAndParse := func() (*models.Connection, error) {
		config := configEntry.Text
		if config == "" {
			return nil, fmt.Errorf("Введите VLESS ссылку")
		}

		conn, err := services.ParseVLESS(config)
		if err != nil {
			return nil, fmt.Errorf("Ошибка парсинга: %v", err)
		}

		if nameEntry.Text != "" {
			conn.Name = nameEntry.Text
		}

		return conn, nil
	}

	configEntry.OnChanged = func(text string) {
		if text != "" {
			conn, err := validateAndParse()
			if err != nil {
				statusLabel.SetText("❌ " + err.Error())
			} else {
				conn.Country, _ = services.GetCountryByIPAPI(conn.Server)
				statusText := fmt.Sprintf("✅ Сервер: %s\n🔒 Безопасность: %s\n📡 Тип: %s",
					conn.Country, conn.Security, conn.Type)
				if conn.SNI != "" {
					statusText += fmt.Sprintf("\n🌐 SNI: %s", conn.SNI)
				}
				statusLabel.SetText(statusText)
			}
		} else {
			statusLabel.SetText("")
		}
	}

	cancelButton.OnTapped = func() { dialog.Hide() }
	addButton.OnTapped = func() {
		conn, err := validateAndParse()
		if err != nil {
			v.app.SendNotification(fyne.NewNotification("Ошибка", err.Error()))
			return
		}

		err = v.components.AddConnection(*conn)
		if err != nil {
			v.app.SendNotification(fyne.NewNotification("Ошибка", "Не удалось сохранить: "+err.Error()))
		} else {
			v.app.SendNotification(fyne.NewNotification("Успех", "Подключение '"+conn.Name+"' добавлено"))
			v.window.SetContent(v.components.GetMainContent())
		}

		dialog.Hide()
	}

	dialog.Show()
}

func (v *VPNApp) copyConnectionLink() {
	connections := v.components.GetConnections()
	if len(connections) == 0 {
		v.app.SendNotification(fyne.NewNotification("Ошибка", "Нет подключений для копирования"))
		return
	}

	if connections[0].Config != "" {
		v.window.Clipboard().SetContent(connections[0].Config)
		v.app.SendNotification(fyne.NewNotification("Успех", "Ссылка скопирована в буфер обмена"))
	} else {
		v.app.SendNotification(fyne.NewNotification("Ошибка", "Нет ссылки для копирования"))
	}
}

func (v *VPNApp) Run() {
	v.app.Run()
}

func (v *VPNApp) UpdateTrayMenu() {
	v.tray.UpdateMenu()
}
