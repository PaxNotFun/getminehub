// Parser de colores ANSI y §Minecraft para la consola del servidor.
package utils

import (
	"image/color"
	"regexp"
	"strings"
)

var ansiColorMap = map[string]color.RGBA{
	"30": {0x22, 0x2f, 0x3e, 0xff}, "31": {0xff, 0x6b, 0x6b, 0xff},
	"32": {0x1d, 0xd1, 0xa1, 0xff}, "33": {0xfe, 0xca, 0x57, 0xff},
	"34": {0x54, 0xa0, 0xff, 0xff}, "35": {0xff, 0x9f, 0xf3, 0xff},
	"36": {0x48, 0xdb, 0xfb, 0xff}, "37": {0xf1, 0xf2, 0xf6, 0xff},
	"90": {0x88, 0x88, 0x88, 0xff}, "91": {0xff, 0x99, 0x99, 0xff},
	"92": {0x55, 0xef, 0xc4, 0xff}, "93": {0xff, 0xea, 0xa7, 0xff},
	"94": {0x74, 0xb9, 0xff, 0xff}, "95": {0xfd, 0x79, 0xa8, 0xff},
	"96": {0x81, 0xec, 0xec, 0xff}, "97": {0xdf, 0xe6, 0xe9, 0xff},
}

var mcColorMap = map[byte]color.RGBA{
	'0': {0x00, 0x00, 0x00, 0xff}, '1': {0x00, 0x00, 0xaa, 0xff},
	'2': {0x00, 0xaa, 0x00, 0xff}, '3': {0x00, 0xaa, 0xaa, 0xff},
	'4': {0xaa, 0x00, 0x00, 0xff}, '5': {0xaa, 0x00, 0xaa, 0xff},
	'6': {0xff, 0xaa, 0x00, 0xff}, '7': {0xaa, 0xaa, 0xaa, 0xff},
	'8': {0x55, 0x55, 0x55, 0xff}, '9': {0x55, 0x55, 0xff, 0xff},
	'a': {0x55, 0xff, 0x55, 0xff}, 'b': {0x55, 0xff, 0xff, 0xff},
	'c': {0xff, 0x55, 0x55, 0xff}, 'd': {0xff, 0x55, 0xff, 0xff},
	'e': {0xff, 0xff, 0x55, 0xff}, 'f': {0xff, 0xff, 0xff, 0xff},
}

var (
	AnsiPattern = regexp.MustCompile(`\x1b\[(\d+(?:;\d+)*)m`)
	McPattern   = regexp.MustCompile(`§([0-9a-fk-or])`)
	combinedPat = regexp.MustCompile(`(\x1b\[(\d+(?:;\d+)*)m)|(§([0-9a-fk-or]))`)
)

// ColoredSegment representa un segmento de texto con un color opcional
type ColoredSegment struct {
	Text  string
	Color *color.RGBA // nil = color por defecto
}

// DefaultConsoleColor es el color por defecto del texto de consola
var DefaultConsoleColor = color.RGBA{0xf1, 0xf5, 0xf9, 0xff}

// ParseANSIAndMinecraft parsea texto con códigos ANSI y §Minecraft
func ParseANSIAndMinecraft(text string) []ColoredSegment {
	var segments []ColoredSegment
	var currentColor *color.RGBA
	lastIndex := 0

	matches := combinedPat.FindAllStringSubmatchIndex(text, -1)
	for _, match := range matches {
		if match[0] > lastIndex {
			seg := text[lastIndex:match[0]]
			if seg != "" {
				var c *color.RGBA
				if currentColor != nil { cc := *currentColor; c = &cc }
				segments = append(segments, ColoredSegment{Text: seg, Color: c})
			}
		}
		lastIndex = match[1]
		fullMatch := text[match[0]:match[1]]

		if strings.HasPrefix(fullMatch, "\x1b") {
			if match[4] >= 0 {
				codes := strings.Split(text[match[4]:match[5]], ";")
				currentColor = parseANSICodes(codes)
			}
		} else if strings.HasPrefix(fullMatch, "§") {
			if match[8] >= 0 && match[9] > match[8] {
				code := text[match[8]:match[9]]
				if len(code) > 0 {
					c := byte(strings.ToLower(code)[0])
					if c == 'r' { currentColor = nil } else if mc, ok := mcColorMap[c]; ok { cc := mc; currentColor = &cc }
				}
			}
		}
	}

	if lastIndex < len(text) {
		seg := text[lastIndex:]
		if seg != "" {
			var c *color.RGBA
			if currentColor != nil { cc := *currentColor; c = &cc }
			segments = append(segments, ColoredSegment{Text: seg, Color: c})
		}
	}
	return segments
}

func parseANSICodes(codes []string) *color.RGBA {
	for _, code := range codes {
		if code == "0" { return nil }
		if c, ok := ansiColorMap[code]; ok { cc := c; return &cc }
	}
	return nil
}

// StripMinecraftColors elimina códigos §X de Minecraft y secuencias ANSI
func StripMinecraftColors(text string) string {
	text = McPattern.ReplaceAllString(text, "")
	text = AnsiPattern.ReplaceAllString(text, "")
	return strings.TrimSpace(text)
}
