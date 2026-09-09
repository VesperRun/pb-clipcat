package main

import (
	"encoding/binary"
	"unicode/utf16"
)

const dropfilesSize = 20

func parseHDROP(data []byte) ([]string, error) {
	if len(data) < dropfilesSize {
		return nil, errf("file list is truncated")
	}
	pFiles := int(binary.LittleEndian.Uint32(data[0:4]))
	if pFiles < dropfilesSize || pFiles > len(data) {
		return nil, errf("file list offset is invalid")
	}
	wide := binary.LittleEndian.Uint32(data[16:20]) != 0
	rest := data[pFiles:]
	if wide {
		return splitUTF16Z(rest), nil
	}
	var paths []string
	for len(rest) > 0 {
		i := 0
		for i < len(rest) && rest[i] != 0 {
			i++
		}
		if i == 0 {
			break
		}
		paths = append(paths, string(rest[:i]))
		rest = rest[i+1:]
	}
	return paths, nil
}

func serializeHDROP(paths []string) []byte {
	var u []uint16
	for _, p := range paths {
		u = append(u, utf16.Encode([]rune(p))...)
		u = append(u, 0)
	}
	u = append(u, 0)
	body := make([]byte, len(u)*2)
	for i, c := range u {
		binary.LittleEndian.PutUint16(body[i*2:], c)
	}
	out := make([]byte, dropfilesSize+len(body))
	binary.LittleEndian.PutUint32(out[0:4], dropfilesSize)
	binary.LittleEndian.PutUint32(out[16:20], 1)
	copy(out[dropfilesSize:], body)
	return out
}

func splitUTF16Z(b []byte) []string {
	if len(b)%2 == 1 {
		b = b[:len(b)-1]
	}
	var paths []string
	var cur []uint16
	for i := 0; i+1 < len(b); i += 2 {
		c := binary.LittleEndian.Uint16(b[i:])
		if c == 0 {
			if len(cur) == 0 {
				break
			}
			paths = append(paths, string(utf16.Decode(cur)))
			cur = cur[:0]
			continue
		}
		cur = append(cur, c)
	}
	return paths
}
