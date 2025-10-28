package main

import (
	"vpn-client/ui"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
)

func main() {
	app := app.NewWithID("com.vpn.client")

	// Установка метаданных для macOS
	app.SetIcon(loadIcon("assets/grey_icon.png"))

	vpnApp := ui.NewVPNApp()
	vpnApp.Run()
}

func loadIcon(path string) fyne.Resource {
	icon, err := fyne.LoadResourceFromPath(path)
	if err != nil {
		return nil
	}
	return icon
}
