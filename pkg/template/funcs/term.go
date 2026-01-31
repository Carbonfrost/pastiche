// Copyright 2026 The Pastiche Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package funcs

import (
	"fmt"
	"os"
	"slices"
	"strings"

	"golang.org/x/term"
)

type TermFuncs struct {
	ColorEnabled bool
}

func NewTermFuncs() *TermFuncs {
	return &TermFuncs{
		ColorEnabled: colorEnabled(),
	}
}

func (t *TermFuncs) Reset() string {
	if t.ColorEnabled {
		return "\x1b[0m"
	}
	return ""
}

func (t *TermFuncs) ResetColor() string {
	return fmt.Sprintf("\x1b[%dm", 39)
}

func (t *TermFuncs) Color(color string, v ...any) (string, error) {
	switch color {
	case "Black":
		return t.Black(v...), nil
	case "Red":
		return t.Red(v...), nil
	case "Green":
		return t.Green(v...), nil
	case "Yellow":
		return t.Yellow(v...), nil
	case "Blue":
		return t.Blue(v...), nil
	case "Magenta":
		return t.Magenta(v...), nil
	case "Cyan":
		return t.Cyan(v...), nil
	case "White":
		return t.White(v...), nil
	case "BrightBlack":
		return t.BrightBlack(v...), nil
	case "BrightRed":
		return t.BrightRed(v...), nil
	case "BrightGreen":
		return t.BrightGreen(v...), nil
	case "BrightYellow":
		return t.BrightYellow(v...), nil
	case "BrightBlue":
		return t.BrightBlue(v...), nil
	case "BrightMagenta":
		return t.BrightMagenta(v...), nil
	case "BrightCyan":
		return t.BrightCyan(v...), nil
	case "BrightWhite":
		return t.BrightWhite(v...), nil
	}
	return "", fmt.Errorf("unknown or unsupported color %q", color)
}

type sprinter func(...any) string

func (t *TermFuncs) styleAtom(style string) (sprinter, bool) {
	switch style {
	case "Bold":
		return t.Bold, true
	case "Dim":
		return t.Dim, true
	case "Italic":
		return t.Italic, true
	case "Underline":
		return t.Underline, true
	case "Blink":
		return t.Blink, true
	case "Reverse":
		return t.Reverse, true
	case "Hidden":
		return t.Hidden, true
	case "Strike":
		return t.Strike, true
	}
	return nil, false
}

func (t *TermFuncs) Style(styles string, a ...any) (string, error) {
	s := strings.Fields(styles)
	switch len(s) {
	case 0:
		return fmt.Sprint(a...), nil
	case 1:
		if s, ok := t.styleAtom(s[0]); ok {
			return s(a...), nil
		}
	default:
		all := make([]sprinter, len(s))
		var ok bool
		for i, style := range s {
			if all[i], ok = t.styleAtom(style); !ok {
				return "", fmt.Errorf("not valid style: %q", styles)
			}
		}
		return sprinterHO(all...)(a...), nil
	}

	return "", fmt.Errorf("not valid style: %q", styles)
}

func sprinterHO(fns ...sprinter) sprinter {
	return func(v ...any) string {
		s := fmt.Sprint(v...)
		for _, f := range slices.Backward(fns) {
			s = f(s)
		}
		return s
	}
}

func (t *TermFuncs) Bold(v ...any) string      { return t.sgr(1, 22, v...) }
func (t *TermFuncs) Dim(v ...any) string       { return t.sgr(2, 22, v...) }
func (t *TermFuncs) Italic(v ...any) string    { return t.sgr(3, 23, v...) }
func (t *TermFuncs) Underline(v ...any) string { return t.sgr(4, 24, v...) }
func (t *TermFuncs) Blink(v ...any) string     { return t.sgr(5, 25, v...) }
func (t *TermFuncs) Reverse(v ...any) string   { return t.sgr(7, 27, v...) }
func (t *TermFuncs) Hidden(v ...any) string    { return t.sgr(8, 28, v...) }
func (t *TermFuncs) Strike(v ...any) string    { return t.sgr(9, 29, v...) }

func (t *TermFuncs) Black(v ...any) string   { return t.sgr(30, 39, v...) }
func (t *TermFuncs) Red(v ...any) string     { return t.sgr(31, 39, v...) }
func (t *TermFuncs) Green(v ...any) string   { return t.sgr(32, 39, v...) }
func (t *TermFuncs) Yellow(v ...any) string  { return t.sgr(33, 39, v...) }
func (t *TermFuncs) Blue(v ...any) string    { return t.sgr(34, 39, v...) }
func (t *TermFuncs) Magenta(v ...any) string { return t.sgr(35, 39, v...) }
func (t *TermFuncs) Cyan(v ...any) string    { return t.sgr(36, 39, v...) }
func (t *TermFuncs) White(v ...any) string   { return t.sgr(37, 39, v...) }

func (t *TermFuncs) BrightBlack(v ...any) string   { return t.sgr(90, 39, v...) }
func (t *TermFuncs) BrightRed(v ...any) string     { return t.sgr(91, 39, v...) }
func (t *TermFuncs) BrightGreen(v ...any) string   { return t.sgr(92, 39, v...) }
func (t *TermFuncs) BrightYellow(v ...any) string  { return t.sgr(93, 39, v...) }
func (t *TermFuncs) BrightBlue(v ...any) string    { return t.sgr(94, 39, v...) }
func (t *TermFuncs) BrightMagenta(v ...any) string { return t.sgr(95, 39, v...) }
func (t *TermFuncs) BrightCyan(v ...any) string    { return t.sgr(96, 39, v...) }
func (t *TermFuncs) BrightWhite(v ...any) string   { return t.sgr(97, 39, v...) }

// TODO Support Background colors

func (t *TermFuncs) sgr(on, off int, v ...any) string {
	if !t.ColorEnabled {
		return fmt.Sprint(v...)
	}

	onExpr := fmt.Sprintf("\x1b[%dm", on)
	if len(v) == 0 {
		return onExpr
	}

	text := fmt.Sprint(v...)
	if len(text) == 0 {
		return ""
	}
	return onExpr +
		text +
		fmt.Sprintf("\x1b[%dm", off)
}

func colorEnabled() bool {
	f := os.Stdout
	return os.Getenv("TERM") != "dumb" && term.IsTerminal(int(f.Fd()))
}
