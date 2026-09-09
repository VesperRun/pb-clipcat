package main

import (
	"bytes"
	"fmt"
	"strings"
	"unicode/utf16"
	"unicode/utf8"
)

func decodeIncomingText(b []byte) string {
	if len(b) >= 2 && b[0] == 0xFF && b[1] == 0xFE {
		return decodeUTF16LE(b[2:])
	}
	if len(b) >= 2 && b[0] == 0xFE && b[1] == 0xFF {
		return decodeUTF16BE(b[2:])
	}
	if len(b) >= 3 && b[0] == 0xEF && b[1] == 0xBB && b[2] == 0xBF {
		return string(b[3:])
	}
	if looksUTF16LE(b) {
		return decodeUTF16LE(b)
	}
	if !utf8.Valid(b) && len(b)%2 == 0 {
		return decodeUTF16LE(b)
	}
	return string(b)
}

func looksUTF16LE(b []byte) bool {
	if len(b) < 4 || len(b)%2 != 0 {
		return false
	}
	nuls := 0
	for i := 1; i < len(b); i += 2 {
		if b[i] == 0 {
			nuls++
		}
	}
	return nuls*2 >= len(b)/2
}

func decodeUTF16LE(b []byte) string {
	if len(b)%2 == 1 {
		b = b[:len(b)-1]
	}
	u := make([]uint16, 0, len(b)/2)
	for i := 0; i+1 < len(b); i += 2 {
		u = append(u, uint16(b[i])|uint16(b[i+1])<<8)
	}
	if n := len(u); n > 0 && u[n-1] == 0 {
		u = u[:n-1]
	}
	return string(utf16.Decode(u))
}

func decodeUTF16BE(b []byte) string {
	if len(b)%2 == 1 {
		b = b[:len(b)-1]
	}
	u := make([]uint16, 0, len(b)/2)
	for i := 0; i+1 < len(b); i += 2 {
		u = append(u, uint16(b[i])<<8|uint16(b[i+1]))
	}
	if n := len(u); n > 0 && u[n-1] == 0 {
		u = u[:n-1]
	}
	return string(utf16.Decode(u))
}

func encodeUTF16LEZ(s string) []byte {
	u := utf16.Encode([]rune(s))
	u = append(u, 0)
	out := make([]byte, len(u)*2)
	for i, c := range u {
		out[i*2] = byte(c)
		out[i*2+1] = byte(c >> 8)
	}
	return out
}

func sniffImage(b []byte) bool {
	if len(b) >= 8 && bytes.Equal(b[:8], []byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n'}) {
		return true
	}
	if len(b) >= 3 && b[0] == 0xff && b[1] == 0xd8 && b[2] == 0xff {
		return true
	}
	if len(b) >= 6 && (bytes.Equal(b[:6], []byte("GIF87a")) || bytes.Equal(b[:6], []byte("GIF89a"))) {
		return true
	}
	if len(b) >= 2 && b[0] == 'B' && b[1] == 'M' {
		return true
	}
	return false
}

func splitPaths(s string, zero bool) []string {
	s = strings.TrimRight(s, "\x00")
	var parts []string
	if zero {
		parts = strings.Split(s, "\x00")
	} else {
		s = toLF(s)
		parts = strings.Split(s, "\n")
	}
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func joinPaths(paths []string, zero bool) string {
	if zero {
		return strings.Join(paths, "\x00") + "\x00"
	}
	return strings.Join(paths, "\n")
}

func errf(format string, args ...any) error {
	return fmt.Errorf("pb: "+format, args...)
}
