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
		// very light lavender background
		return color.NRGBA{R: 250, G: 245, B: 255, A: 255}
	case theme.ColorNameForeground:
		return color.NRGBA{R: 30, G: 30, B: 35, A: 255}
	case theme.ColorNamePrimary:
		// neon pink accent
		return color.NRGBA{R: 255, G: 45, B: 149, A: 255}
	case theme.ColorNameButton:
		// soft purple button background for light theme
		return color.NRGBA{R: 210, G: 190, B: 255, A: 255}
	case theme.ColorNameInputBackground:
		return color.NRGBA{R: 255, G: 255, B: 255, A: 255}
	case theme.ColorNameDisabled:
		return color.NRGBA{R: 200, G: 200, B: 200, A: 255}
	case theme.ColorNamePlaceHolder:
		return color.NRGBA{R: 150, G: 130, B: 180, A: 255}
	case theme.ColorNameHover:
		// subtle purple hover tint
		return color.NRGBA{R: 245, G: 230, B: 255, A: 255}
	case theme.ColorNameFocus:
		return color.NRGBA{R: 255, G: 220, B: 245, A: 255}
	case theme.ColorNamePressed:
		return color.NRGBA{R: 230, G: 170, B: 210, A: 255}
	default:
		return theme.LightTheme().Color(name, theme.VariantLight)
	}
}

func (m *MinimalTheme) darkColor(name fyne.ThemeColorName) color.Color {
	switch name {
	case theme.ColorNameBackground:
		// deep purple background matching the icon tone
		return color.NRGBA{R: 29, G: 29, B: 29, A: 255}
	case theme.ColorNameForeground:
		return color.NRGBA{R: 235, G: 235, B: 240, A: 255}
	case theme.ColorNamePrimary:
		// neon pink accent stays strong in dark theme
		return color.NRGBA{R: 255, G: 45, B: 149, A: 255}
	case theme.ColorNameButton:
		// deep purple button for contrast
		return color.NRGBA{R: 70, G: 20, B: 90, A: 255}
	case theme.ColorNameInputBackground:
		return color.NRGBA{R: 38, G: 24, B: 40, A: 255}
	case theme.ColorNameDisabled:
		return color.NRGBA{R: 90, G: 90, B: 100, A: 255}
	case theme.ColorNamePlaceHolder:
		return color.NRGBA{R: 150, G: 120, B: 160, A: 255}
	case theme.ColorNameHover:
		// subtle purple tint on hover
		return color.NRGBA{R: 80, G: 30, B: 110, A: 255}
	case theme.ColorNameFocus:
		return color.NRGBA{R: 100, G: 40, B: 130, A: 255}
	case theme.ColorNamePressed:
		return color.NRGBA{R: 140, G: 50, B: 160, A: 255}
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
