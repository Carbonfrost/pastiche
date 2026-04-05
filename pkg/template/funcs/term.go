// Copyright 2026 The Pastiche Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package funcs

import (
	"fmt"
	"os"
	"regexp"
	"slices"
	"strings"

	"golang.org/x/term"
)

type TermFuncs struct {
	ColorEnabled bool
}

// rule maps a regex pattern string to a transformation function.
type rule struct {
	Pattern string
	Fn      sprinter
}

// compiledRules holds the combined regex and dispatch table.
type compiledRules struct {
	re        *regexp.Regexp
	groupToFn map[int]sprinter
}

func NewTermFuncs() *TermFuncs {
	return &TermFuncs{
		ColorEnabled: colorEnabled(),
	}
}

func (t *TermFuncs) Self() any {
	return t
}

func (t *TermFuncs) Reset() string {
	if t.ColorEnabled {
		return "\x1b[0m"
	}
	return ""
}

func (t *TermFuncs) ResetColor(v ...any) string {
	return fmt.Sprintf("\x1b[%dm", 39) + fmt.Sprint(v...)
}

func (t *TermFuncs) Colorize(str string, colorpattern ...string) (string, error) {
	if len(colorpattern) == 0 {
		return str, nil
	}

	rules := make([]rule, len(colorpattern))
	for i, cp := range colorpattern {
		color, pattern, ok := strings.Cut(cp, ":")
		if !ok {
			// TODO Probably need a better definition of what to do here
			continue
		}

		fn, err := t.colorSprinter(color)
		if err != nil {
			return str, err
		}

		rules[i] = rule{
			Pattern: pattern,
			Fn:      fn,
		}
	}

	cr, err := compileRules(rules)
	if err != nil {
		panic(err)
	}

	return cr.Apply(str), nil
}

func compileRules(rules []rule) (*compiledRules, error) {
	var parts []string
	groupToFn := make(map[int]sprinter)

	for i, r := range rules {
		groupName := fmt.Sprintf("R%d", i)
		parts = append(parts, fmt.Sprintf("(?P<%s>%s)", groupName, r.Pattern))
	}

	combined := strings.Join(parts, "|")
	re, err := regexp.Compile(combined)
	if err != nil {
		return nil, err
	}

	// Map capture group index to function
	subexpNames := re.SubexpNames()
	for idx, name := range subexpNames {
		if name == "" {
			continue
		}
		var ruleIndex int
		_, err := fmt.Sscanf(name, "R%d", &ruleIndex)
		if err != nil {
			continue
		}
		groupToFn[idx] = rules[ruleIndex].Fn
	}

	return &compiledRules{
		re:        re,
		groupToFn: groupToFn,
	}, nil
}

// Apply applies the compiled rules to the input string.
func (cr *compiledRules) Apply(input string) string {
	var result strings.Builder
	last := 0

	matches := cr.re.FindAllStringSubmatchIndex(input, -1)

	for _, m := range matches {
		start, end := m[0], m[1]
		result.WriteString(input[last:start])

		for groupIdx, fn := range cr.groupToFn {
			gStart := m[2*groupIdx]
			gEnd := m[2*groupIdx+1]
			if gStart != -1 && gEnd != -1 {
				result.WriteString(fn(input[gStart:gEnd]))
				break
			}
		}

		last = end
	}

	result.WriteString(input[last:])
	return result.String()
}

func (t *TermFuncs) Color(color string, v ...any) (string, error) {
	sp, err := t.colorSprinter(color)
	if err != nil {
		return "", err
	}
	return sp(v...), nil
}

func (t *TermFuncs) colorSprinter(color string) (sprinter, error) {
	switch strings.ToLower(color) {
	case "black":
		return t.Black, nil
	case "red":
		return t.Red, nil
	case "green":
		return t.Green, nil
	case "yellow":
		return t.Yellow, nil
	case "blue":
		return t.Blue, nil
	case "magenta":
		return t.Magenta, nil
	case "cyan":
		return t.Cyan, nil
	case "white":
		return t.White, nil
	case "brightblack":
		return t.BrightBlack, nil
	case "brightred":
		return t.BrightRed, nil
	case "brightgreen":
		return t.BrightGreen, nil
	case "brightyellow":
		return t.BrightYellow, nil
	case "brightblue":
		return t.BrightBlue, nil
	case "brightmagenta":
		return t.BrightMagenta, nil
	case "brightcyan":
		return t.BrightCyan, nil
	case "brightwhite":
		return t.BrightWhite, nil
	case "resetcolor":
		return t.ResetColor, nil
	}
	return nil, fmt.Errorf("unknown or unsupported color %q", color)
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
