package main

import "github.com/gdamore/tcell/v2"

// Theme holds all styled color pairs used by the editor.
type Theme struct {
	Name       string
	Normal     tcell.Style // text area
	LineNum    tcell.Style // gutter line numbers
	Selection  tcell.Style // selected text
	Header     tcell.Style // top bar
	HeaderAcct tcell.Style // top bar accent (app name)
	Hints      tcell.Style // bottom shortcut bar
	HintsKey   tcell.Style // ^KEY highlight in hints
	MsgBar     tcell.Style // message bar (normal)
	MsgBarErr  tcell.Style // message bar (error)
	FindHL     tcell.Style // search match highlight
	FindCur    tcell.Style // current search match
	// syntax highlighting
	HLKeyword tcell.Style
	HLType    tcell.Style
	HLBuiltin tcell.Style
	HLString  tcell.Style
	HLComment tcell.Style
	HLNumber  tcell.Style
}

func hex(v int32) tcell.Color { return tcell.NewHexColor(v) }

func mkStyle(fg, bg int32) tcell.Style {
	return tcell.StyleDefault.Foreground(hex(fg)).Background(hex(bg))
}

func DefaultThemes() []*Theme {
	return []*Theme{
		ThemeGreen(),
		ThemeAmber(),
		ThemeBlue(),
		ThemeMono(),
		ThemeLight(),
	}
}

// ThemeGreen is the default HostLab terminal theme.
func ThemeGreen() *Theme {
	bg := int32(0x0a140a)
	return &Theme{
		Name:       "Green  (HostLab default)",
		Normal:     mkStyle(0xb8dfb8, bg),
		LineNum:    mkStyle(0x2d6b2d, bg),
		Selection:  mkStyle(0xd0f0d0, 0x1a4a1a),
		Header:     mkStyle(0x4ec94e, 0x071407),
		HeaderAcct: tcell.StyleDefault.Foreground(hex(0x8eff8e)).Background(hex(0x071407)).Bold(true),
		Hints:      mkStyle(0x2d6b2d, 0x071407),
		HintsKey:   mkStyle(0x4ec94e, 0x071407),
		MsgBar:     mkStyle(0x4ec94e, 0x071407),
		MsgBarErr:  mkStyle(0xff6060, 0x1a0000),
		FindHL:     mkStyle(0x101010, 0xc4a010),
		FindCur:    mkStyle(0x101010, 0xffcc00),
		HLKeyword:  tcell.StyleDefault.Foreground(hex(0x88ff88)).Background(hex(bg)).Bold(true),
		HLType:     mkStyle(0x55ccaa, bg),
		HLBuiltin:  mkStyle(0x66ddcc, bg),
		HLString:   mkStyle(0xccaa44, bg),
		HLComment:  mkStyle(0x3a6a3a, bg),
		HLNumber:   mkStyle(0x80ccff, bg),
	}
}

func ThemeAmber() *Theme {
	bg := int32(0x110d00)
	return &Theme{
		Name:       "Amber",
		Normal:     mkStyle(0xe0c87a, bg),
		LineNum:    mkStyle(0x6a4e10, bg),
		Selection:  mkStyle(0xf0dfa0, 0x3a2a00),
		Header:     mkStyle(0xc4940a, 0x0a0700),
		HeaderAcct: tcell.StyleDefault.Foreground(hex(0xffcc44)).Background(hex(0x0a0700)).Bold(true),
		Hints:      mkStyle(0x6a4e10, 0x0a0700),
		HintsKey:   mkStyle(0xc4940a, 0x0a0700),
		MsgBar:     mkStyle(0xc4940a, 0x0a0700),
		MsgBarErr:  mkStyle(0xff6060, 0x1a0000),
		FindHL:     mkStyle(0x101010, 0x4a8a2a),
		FindCur:    mkStyle(0x101010, 0x88cc44),
		HLKeyword:  tcell.StyleDefault.Foreground(hex(0xffe066)).Background(hex(bg)).Bold(true),
		HLType:     mkStyle(0xd48c44, bg),
		HLBuiltin:  mkStyle(0xc0a040, bg),
		HLString:   mkStyle(0x88cc66, bg),
		HLComment:  mkStyle(0x6a4a18, bg),
		HLNumber:   mkStyle(0x80ccee, bg),
	}
}

func ThemeBlue() *Theme {
	bg := int32(0x080d18)
	return &Theme{
		Name:       "Blue",
		Normal:     mkStyle(0xa8c8e8, bg),
		LineNum:    mkStyle(0x2a4a8a, bg),
		Selection:  mkStyle(0xd0e4f8, 0x1a2a5a),
		Header:     mkStyle(0x5a9ae5, 0x050a12),
		HeaderAcct: tcell.StyleDefault.Foreground(hex(0x90c8ff)).Background(hex(0x050a12)).Bold(true),
		Hints:      mkStyle(0x2a4a8a, 0x050a12),
		HintsKey:   mkStyle(0x5a9ae5, 0x050a12),
		MsgBar:     mkStyle(0x5a9ae5, 0x050a12),
		MsgBarErr:  mkStyle(0xff6060, 0x1a0000),
		FindHL:     mkStyle(0x101010, 0xc49010),
		FindCur:    mkStyle(0x101010, 0xffcc00),
		HLKeyword:  tcell.StyleDefault.Foreground(hex(0x88aaff)).Background(hex(bg)).Bold(true),
		HLType:     mkStyle(0x44cccc, bg),
		HLBuiltin:  mkStyle(0x66bbee, bg),
		HLString:   mkStyle(0x88cc66, bg),
		HLComment:  mkStyle(0x3a5878, bg),
		HLNumber:   mkStyle(0xffcc66, bg),
	}
}

func ThemeMono() *Theme {
	bg := int32(0x0a0a0a)
	return &Theme{
		Name:       "Monochrome",
		Normal:     mkStyle(0xd0d0d0, bg),
		LineNum:    mkStyle(0x3a3a3a, bg),
		Selection:  mkStyle(0xf0f0f0, 0x2a2a2a),
		Header:     mkStyle(0xf0f0f0, 0x050505),
		HeaderAcct: tcell.StyleDefault.Foreground(hex(0xffffff)).Background(hex(0x050505)).Bold(true),
		Hints:      mkStyle(0x3a3a3a, 0x050505),
		HintsKey:   mkStyle(0xc0c0c0, 0x050505),
		MsgBar:     mkStyle(0xd0d0d0, 0x050505),
		MsgBarErr:  mkStyle(0xff6060, 0x1a0000),
		FindHL:     mkStyle(0x101010, 0xb0b010),
		FindCur:    mkStyle(0x101010, 0xffff00),
		HLKeyword:  tcell.StyleDefault.Foreground(hex(0xffffff)).Background(hex(bg)).Bold(true),
		HLType:     mkStyle(0xcccccc, bg),
		HLBuiltin:  mkStyle(0xb0b0b0, bg),
		HLString:   mkStyle(0x888888, bg),
		HLComment:  mkStyle(0x505050, bg),
		HLNumber:   mkStyle(0xe0e0e0, bg),
	}
}

func ThemeLight() *Theme {
	bg := int32(0xf0ede6)
	return &Theme{
		Name:       "Light",
		Normal:     mkStyle(0x1a1a16, bg),
		LineNum:    mkStyle(0xa0a09a, bg),
		Selection:  mkStyle(0x1a1a16, 0xc8e0c8),
		Header:     mkStyle(0x1a1a16, 0xd8d5ce),
		HeaderAcct: tcell.StyleDefault.Foreground(hex(0x0a0a08)).Background(hex(0xd8d5ce)).Bold(true),
		Hints:      mkStyle(0xa0a09a, 0xd8d5ce),
		HintsKey:   mkStyle(0x1a1a16, 0xd8d5ce),
		MsgBar:     mkStyle(0x1a1a16, 0xd8d5ce),
		MsgBarErr:  mkStyle(0xcc2020, 0xffdede),
		FindHL:     mkStyle(0x1a1a16, 0xf0d060),
		FindCur:    mkStyle(0x101010, 0xffcc00),
		HLKeyword:  tcell.StyleDefault.Foreground(hex(0x0055cc)).Background(hex(bg)).Bold(true),
		HLType:     mkStyle(0x005588, bg),
		HLBuiltin:  mkStyle(0x007766, bg),
		HLString:   mkStyle(0x226600, bg),
		HLComment:  mkStyle(0x909080, bg),
		HLNumber:   mkStyle(0x882200, bg),
	}
}
