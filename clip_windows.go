//go:build windows

package main

import (
	"fmt"
	"path/filepath"
	"syscall"
	"time"
	"unsafe"
)

var (
	user32   = syscall.NewLazyDLL("user32.dll")
	kernel32 = syscall.NewLazyDLL("kernel32.dll")

	procOpenClipboard              = user32.NewProc("OpenClipboard")
	procCloseClipboard             = user32.NewProc("CloseClipboard")
	procEmptyClipboard             = user32.NewProc("EmptyClipboard")
	procGetClipboardData           = user32.NewProc("GetClipboardData")
	procSetClipboardData           = user32.NewProc("SetClipboardData")
	procIsClipboardFormatAvailable = user32.NewProc("IsClipboardFormatAvailable")
	procRegisterClipboardFormatW   = user32.NewProc("RegisterClipboardFormatW")
	procEnumClipboardFormats       = user32.NewProc("EnumClipboardFormats")
	procGetClipboardFormatNameW    = user32.NewProc("GetClipboardFormatNameW")

	procGlobalAlloc  = kernel32.NewProc("GlobalAlloc")
	procGlobalLock   = kernel32.NewProc("GlobalLock")
	procGlobalUnlock = kernel32.NewProc("GlobalUnlock")
	procGlobalSize   = kernel32.NewProc("GlobalSize")
	procGlobalFree   = kernel32.NewProc("GlobalFree")
)

const (
	cfText         = 1
	cfBitmap       = 2
	cfDIB          = 8
	cfOEMText      = 7
	cfUnicodeText  = 13
	cfHDROP        = 15
	cfDIBV5        = 17
	gmemMoveable   = 0x0002
	clipboardRetry = 50
)

var knownFormats = map[uint32]string{
	cfText:        "CF_TEXT",
	cfBitmap:      "CF_BITMAP",
	3:             "CF_METAFILEPICT",
	4:             "CF_SYLK",
	5:             "CF_DIF",
	6:             "CF_TIFF",
	cfOEMText:     "CF_OEMTEXT",
	cfDIB:         "CF_DIB",
	9:             "CF_PALETTE",
	10:            "CF_PENDATA",
	11:            "CF_RIFF",
	12:            "CF_WAVE",
	cfUnicodeText: "CF_UNICODETEXT",
	14:            "CF_ENHMETAFILE",
	cfHDROP:       "CF_HDROP",
	16:            "CF_LOCALE",
	cfDIBV5:       "CF_DIBV5",
}

func openClipboard() error {
	var last error
	for i := 0; i < clipboardRetry; i++ {
		r, _, err := procOpenClipboard.Call(0)
		if r != 0 {
			return nil
		}
		last = err
		time.Sleep(10 * time.Millisecond)
	}
	return errf("could not open clipboard: %v", last)
}

func closeClipboard() {
	procCloseClipboard.Call()
}

func emptyClipboard() error {
	r, _, err := procEmptyClipboard.Call()
	if r == 0 {
		return errf("could not empty clipboard: %v", err)
	}
	return nil
}

func withClipboard(empty bool, fn func() error) error {
	if err := openClipboard(); err != nil {
		return err
	}
	defer closeClipboard()
	if empty {
		if err := emptyClipboard(); err != nil {
			return err
		}
	}
	return fn()
}

func registerFormat(name string) (uint32, error) {
	p, err := syscall.UTF16PtrFromString(name)
	if err != nil {
		return 0, err
	}
	r, _, callErr := procRegisterClipboardFormatW.Call(uintptr(unsafe.Pointer(p)))
	if r == 0 {
		return 0, errf("register format %s: %v", name, callErr)
	}
	return uint32(r), nil
}

func hasFormat(format uint32) bool {
	r, _, _ := procIsClipboardFormatAvailable.Call(uintptr(format))
	return r != 0
}

func getBytes(format uint32) ([]byte, error) {
	h, _, err := procGetClipboardData.Call(uintptr(format))
	if h == 0 {
		return nil, errf("get clipboard data: %v", err)
	}
	ptr, _, err := procGlobalLock.Call(h)
	if ptr == 0 {
		return nil, errf("lock clipboard: %v", err)
	}
	defer procGlobalUnlock.Call(h)
	size, _, _ := procGlobalSize.Call(h)
	if size == 0 {
		return []byte{}, nil
	}
	src := unsafe.Slice((*byte)(unsafe.Pointer(ptr)), int(size))
	out := make([]byte, len(src))
	copy(out, src)
	return out, nil
}

func setBytes(format uint32, data []byte) error {
	h, _, err := procGlobalAlloc.Call(gmemMoveable, uintptr(len(data)))
	if h == 0 {
		return errf("alloc clipboard: %v", err)
	}
	ptr, _, err := procGlobalLock.Call(h)
	if ptr == 0 {
		procGlobalFree.Call(h)
		return errf("lock clipboard: %v", err)
	}
	dst := unsafe.Slice((*byte)(unsafe.Pointer(ptr)), len(data))
	copy(dst, data)
	procGlobalUnlock.Call(h)
	r, _, err := procSetClipboardData.Call(uintptr(format), h)
	if r == 0 {
		procGlobalFree.Call(h)
		return errf("set clipboard: %v", err)
	}
	return nil
}

func listFormats() ([]string, error) {
	var names []string
	err := withClipboard(false, func() error {
		var format uint32
		for {
			r, _, _ := procEnumClipboardFormats.Call(uintptr(format))
			format = uint32(r)
			if format == 0 {
				break
			}
			names = append(names, formatName(format))
		}
		return nil
	})
	return names, err
}

func formatName(format uint32) string {
	if s, ok := knownFormats[format]; ok {
		return s
	}
	var buf [256]uint16
	n, _, _ := procGetClipboardFormatNameW.Call(uintptr(format), uintptr(unsafe.Pointer(&buf[0])), uintptr(len(buf)))
	if n > 0 {
		return syscall.UTF16ToString(buf[:n])
	}
	return fmt.Sprintf("FORMAT_%d", format)
}

func clearClipboard() error {
	return withClipboard(true, func() error { return nil })
}

func pasteText() (string, error) {
	var s string
	err := withClipboard(false, func() error {
		if !hasFormat(cfUnicodeText) && !hasFormat(cfText) {
			return errf("no text on clipboard")
		}
		if hasFormat(cfUnicodeText) {
			b, err := getBytes(cfUnicodeText)
			if err != nil {
				return err
			}
			s = decodeUTF16LE(b)
			return nil
		}
		b, err := getBytes(cfText)
		if err != nil {
			return err
		}
		s = string(bytesTrimZ(b))
		return nil
	})
	return s, err
}

func pasteHTML() (string, error) {
	var s string
	err := withClipboard(false, func() error {
		fmtID, err := registerFormat("HTML Format")
		if err != nil {
			return err
		}
		if !hasFormat(fmtID) {
			return errf("no HTML on clipboard")
		}
		b, err := getBytes(fmtID)
		if err != nil {
			return err
		}
		s = unwrapHTML(string(bytesTrimZ(b)))
		return nil
	})
	return s, err
}

func pasteFiles() ([]string, error) {
	var paths []string
	err := withClipboard(false, func() error {
		if !hasFormat(cfHDROP) {
			return errf("no file list on clipboard")
		}
		b, err := getBytes(cfHDROP)
		if err != nil {
			return err
		}
		paths, err = parseHDROP(b)
		return err
	})
	return paths, err
}

func pastePNG() ([]byte, int, int, error) {
	var pngBytes []byte
	var w, h int
	err := withClipboard(false, func() error {
		if b, err := getRegisteredImage("PNG"); err == nil {
			pngBytes, w, h, err = imageToPNG(b)
			return err
		}
		if b, err := getRegisteredImage("image/png"); err == nil {
			pngBytes, w, h, err = imageToPNG(b)
			return err
		}
		if hasFormat(cfDIB) {
			b, err := getBytes(cfDIB)
			if err != nil {
				return err
			}
			pngBytes, w, h, err = dibToPNG(b)
			return err
		}
		if hasFormat(cfDIBV5) {
			b, err := getBytes(cfDIBV5)
			if err != nil {
				return err
			}
			pngBytes, w, h, err = dibToPNG(b)
			return err
		}
		return errf("no image on clipboard")
	})
	return pngBytes, w, h, err
}

func getRegisteredImage(name string) ([]byte, error) {
	id, err := registerFormat(name)
	if err != nil {
		return nil, err
	}
	if !hasFormat(id) {
		return nil, errf("format %s missing", name)
	}
	return getBytes(id)
}

func clipboardKind() (Flavor, error) {
	var f Flavor
	err := withClipboard(false, func() error {
		if hasFormat(cfUnicodeText) || hasFormat(cfText) {
			f = FlavorText
			return nil
		}
		if hasFormat(cfHDROP) {
			f = FlavorFiles
			return nil
		}
		if hasImageLocked() {
			f = FlavorImage
			return nil
		}
		return errf("clipboard is empty")
	})
	return f, err
}

func hasImageLocked() bool {
	if hasFormat(cfDIB) || hasFormat(cfDIBV5) || hasFormat(cfBitmap) {
		return true
	}
	if id, err := registerFormat("PNG"); err == nil && hasFormat(id) {
		return true
	}
	if id, err := registerFormat("image/png"); err == nil && hasFormat(id) {
		return true
	}
	return false
}

func copyText(s string) error {
	return withClipboard(true, func() error {
		return setBytes(cfUnicodeText, encodeUTF16LEZ(s))
	})
}

func copyHTML(fragment string) error {
	payload := wrapHTML(fragment)
	text := stripTags(fragment)
	return withClipboard(true, func() error {
		id, err := registerFormat("HTML Format")
		if err != nil {
			return err
		}
		if err := setBytes(id, append([]byte(payload), 0)); err != nil {
			return err
		}
		return setBytes(cfUnicodeText, encodeUTF16LEZ(text))
	})
}

func copyFiles(paths []string) error {
	if len(paths) == 0 {
		return errf("no files to copy")
	}
	abs := make([]string, 0, len(paths))
	for _, p := range paths {
		a, err := filepath.Abs(p)
		if err != nil {
			a = p
		}
		abs = append(abs, a)
	}
	data := serializeHDROP(abs)
	return withClipboard(true, func() error {
		return setBytes(cfHDROP, data)
	})
}

func copyPNG(pngBytes []byte) error {
	dib, err := pngToDIB(pngBytes)
	if err != nil {
		return err
	}
	return withClipboard(true, func() error {
		id, err := registerFormat("PNG")
		if err == nil {
			if err := setBytes(id, pngBytes); err != nil {
				return err
			}
		}
		return setBytes(cfDIB, dib)
	})
}

func bytesTrimZ(b []byte) []byte {
	i := len(b)
	for i > 0 && b[i-1] == 0 {
		i--
	}
	return b[:i]
}

func imageHint() string {
	b, w, h, err := pastePNG()
	if err != nil {
		return "pb: clipboard holds an image\n     write it with:  pb -o shot.png\n"
	}
	if w == 0 || h == 0 {
		w, h = pngSize(b)
	}
	return fmt.Sprintf("pb: clipboard holds an image (%dx%d PNG)\n     write it with:  pb -o shot.png\n", w, h)
}
