package main

import (
	"encoding/binary"
	"image"
	"image/color"
	"image/png"
	"bytes"
	"testing"
)

func TestPNGDIBRoundTrip(t *testing.T) {
	img := image.NewNRGBA(image.Rect(0, 0, 2, 1))
	img.SetNRGBA(0, 0, color.NRGBA{255, 0, 0, 255})
	img.SetNRGBA(1, 0, color.NRGBA{0, 255, 0, 255})
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	dib, err := pngToDIB(buf.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	out, w, h, err := dibToPNG(dib)
	if err != nil {
		t.Fatal(err)
	}
	if w != 2 || h != 1 {
		t.Fatalf("size %dx%d", w, h)
	}
	got, err := png.Decode(bytes.NewReader(out))
	if err != nil {
		t.Fatal(err)
	}
	r, _, _, _ := got.At(0, 0).RGBA()
	if r>>8 < 200 {
		t.Fatalf("expected red pixel, got %v", got.At(0, 0))
	}
}

func TestSniffAndUTF16(t *testing.T) {
	if !sniffImage([]byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n'}) {
		t.Fatal("png sniff")
	}
	s := decodeIncomingText([]byte{'h', 0, 'i', 0})
	if s != "hi" {
		t.Fatalf("utf16 %q", s)
	}
	if toCRLF("a\nb") != "a\r\nb" {
		t.Fatal("crlf")
	}
}

func TestTinyDIBHeader(t *testing.T) {
	dib := make([]byte, 40+4)
	binary.LittleEndian.PutUint32(dib[0:4], 40)
	binary.LittleEndian.PutUint32(dib[4:8], 1)
	binary.LittleEndian.PutUint32(dib[8:12], 1)
	binary.LittleEndian.PutUint16(dib[12:14], 1)
	binary.LittleEndian.PutUint16(dib[14:16], 32)
	dib[40], dib[41], dib[42], dib[43] = 0, 0, 255, 255 // red in BGRA
	_, w, h, err := dibToPNG(dib)
	if err != nil {
		t.Fatal(err)
	}
	if w != 1 || h != 1 {
		t.Fatalf("%dx%d", w, h)
	}
}
