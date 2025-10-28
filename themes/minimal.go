package themes

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
)

type MinimalTheme struct {
	isDark bool
}

func NewMinimalTheme(isDark bool) *MinimalTheme {
	return &MinimalTheme{isDark: isDark}
}

func (m *MinimalTheme) Color(name fyne.ThemeColorName, variant fyne.ThemeVariant) color.Color {
	if m.isDark {
		return m.darkColor(name)
	}
	return m.lightColor(name)
}

func (m *MinimalTheme) lightColor(name fyne.ThemeColorName) color.Color {
	switch name {
	case theme.ColorNameBackground:
		return color.NRGBA{R: 250, G: 250, B: 250, A: 255}
	case theme.ColorNameForeground:
		return color.NRGBA{R: 40, G: 40, B: 40, A: 255}
	case theme.ColorNamePrimary:
		return color.NRGBA{R: 80, G: 80, B: 80, A: 255}
	case theme.ColorNameButton:
		return color.NRGBA{R: 240, G: 240, B: 240, A: 255}
	case theme.ColorNameInputBackground:
		return color.NRGBA{R: 255, G: 255, B: 255, A: 255}
	case theme.ColorNameDisabled:
		return color.NRGBA{R: 200, G: 200, B: 200, A: 255}
	case theme.ColorNamePlaceHolder:
		return color.NRGBA{R: 160, G: 160, B: 160, A: 255}
	case theme.ColorNameHover:
		return color.NRGBA{R: 230, G: 230, B: 230, A: 255}
	case theme.ColorNameFocus:
		return color.NRGBA{R: 220, G: 220, B: 220, A: 255}
	case theme.ColorNamePressed:
		return color.NRGBA{R: 210, G: 210, B: 210, A: 255}
	default:
		return theme.LightTheme().Color(name, theme.VariantLight)
	}
}

func (m *MinimalTheme) darkColor(name fyne.ThemeColorName) color.Color {
	switch name {
	case theme.ColorNameBackground:
		return color.NRGBA{R: 49, G: 49, B: 49, A: 255}
	case theme.ColorNameForeground:
		return color.NRGBA{R: 220, G: 220, B: 220, A: 255}
	case theme.ColorNamePrimary:
		return color.NRGBA{R: 180, G: 180, B: 180, A: 255}
	case theme.ColorNameButton:
		return color.NRGBA{R: 60, G: 60, B: 65, A: 255}
	case theme.ColorNameInputBackground:
		return color.NRGBA{R: 50, G: 50, B: 55, A: 255}
	case theme.ColorNameDisabled:
		return color.NRGBA{R: 100, G: 100, B: 100, A: 255}
	case theme.ColorNamePlaceHolder:
		return color.NRGBA{R: 140, G: 140, B: 140, A: 255}
	case theme.ColorNameHover:
		return color.NRGBA{R: 70, G: 70, B: 75, A: 255}
	case theme.ColorNameFocus:
		return color.NRGBA{R: 80, G: 80, B: 85, A: 255}
	case theme.ColorNamePressed:
		return color.NRGBA{R: 90, G: 90, B: 95, A: 255}
	default:
		return theme.DarkTheme().Color(name, theme.VariantDark)
	}
}

func (m *MinimalTheme) Icon(name fyne.ThemeIconName) fyne.Resource {
	return theme.DefaultTheme().Icon(name)
}

func (m *MinimalTheme) Font(style fyne.TextStyle) fyne.Resource {
	return theme.DefaultTheme().Font(style)
}

func (m *MinimalTheme) Size(name fyne.ThemeSizeName) float32 {
	if name == theme.SizeNamePadding {
		return 12
	}
	if name == theme.SizeNameText {
		return 14
	}
	return theme.DefaultTheme().Size(name)
}
