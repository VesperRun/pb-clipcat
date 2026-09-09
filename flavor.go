package main

import (
	"path/filepath"
	"strings"
)

type Flavor int

const (
	FlavorAuto Flavor = iota
	FlavorText
	FlavorHTML
	FlavorFiles
	FlavorImage
)

func (f Flavor) String() string {
	switch f {
	case FlavorText:
		return "text"
	case FlavorHTML:
		return "html"
	case FlavorFiles:
		return "files"
	case FlavorImage:
		return "image"
	default:
		return "auto"
	}
}

func flavorFromExt(name string) Flavor {
	switch strings.ToLower(filepath.Ext(name)) {
	case ".png", ".jpg", ".jpeg", ".gif", ".bmp", ".webp":
		return FlavorImage
	case ".html", ".htm":
		return FlavorHTML
	default:
		return FlavorText
	}
}

func toLF(s string) string {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	return strings.ReplaceAll(s, "\r", "\n")
}

func toCRLF(s string) string {
	return strings.ReplaceAll(toLF(s), "\n", "\r\n")
}
