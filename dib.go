package main

import (
	"bytes"
	"encoding/binary"
	"image"
	"image/color"
	"image/draw"
	_ "image/gif"
	_ "image/jpeg"
	"image/png"

	"golang.org/x/image/bmp"
)

func dibToPNG(dib []byte) ([]byte, int, int, error) {
	if len(dib) < 40 {
		return nil, 0, 0, errf("DIB is truncated")
	}
	biSize := binary.LittleEndian.Uint32(dib[0:4])
	width := int(int32(binary.LittleEndian.Uint32(dib[4:8])))
	height := int(int32(binary.LittleEndian.Uint32(dib[8:12])))
	if width <= 0 {
		return nil, 0, 0, errf("DIB width is invalid")
	}

	bmpFile := dibToBMP(dib)
	img, err := bmp.Decode(bytes.NewReader(bmpFile))
	if err != nil {
		img, err = decodeRGBDIB(dib)
		if err != nil {
			return nil, 0, 0, err
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, 0, 0, err
	}
	h := height
	if h < 0 {
		h = -h
	}
	_ = biSize
	b := img.Bounds()
	return buf.Bytes(), b.Dx(), b.Dy(), nil
}

func dibToBMP(dib []byte) []byte {
	off := 14 + pixelOffset(dib)
	out := make([]byte, 14+len(dib))
	out[0], out[1] = 'B', 'M'
	binary.LittleEndian.PutUint32(out[2:6], uint32(len(out)))
	binary.LittleEndian.PutUint32(out[10:14], uint32(off))
	copy(out[14:], dib)
	return out
}

func pixelOffset(dib []byte) int {
	if len(dib) < 40 {
		return 40
	}
	biSize := int(binary.LittleEndian.Uint32(dib[0:4]))
	bitCount := binary.LittleEndian.Uint16(dib[14:16])
	compression := binary.LittleEndian.Uint32(dib[16:20])
	clrUsed := int(binary.LittleEndian.Uint32(dib[32:36]))
	off := biSize
	if compression == 3 || compression == 6 { // BI_BITFIELDS / BI_ALPHABITFIELDS
		if biSize == 40 {
			off += 12
		}
	}
	if bitCount <= 8 {
		n := clrUsed
		if n == 0 {
			n = 1 << bitCount
		}
		off += n * 4
	}
	if off < 40 || off > len(dib) {
		return biSize
	}
	return off
}

func decodeRGBDIB(dib []byte) (image.Image, error) {
	if len(dib) < 40 {
		return nil, errf("DIB is truncated")
	}
	width := int(int32(binary.LittleEndian.Uint32(dib[4:8])))
	heightSigned := int(int32(binary.LittleEndian.Uint32(dib[8:12])))
	bitCount := binary.LittleEndian.Uint16(dib[14:16])
	compression := binary.LittleEndian.Uint32(dib[16:20])
	if compression != 0 && compression != 3 {
		return nil, errf("unsupported DIB compression")
	}
	topDown := heightSigned < 0
	height := heightSigned
	if height < 0 {
		height = -height
	}
	off := pixelOffset(dib)
	if width <= 0 || height <= 0 || off >= len(dib) {
		return nil, errf("DIB geometry is invalid")
	}
	dst := image.NewNRGBA(image.Rect(0, 0, width, height))
	src := dib[off:]
	switch bitCount {
	case 32:
		row := width * 4
		for y := 0; y < height; y++ {
			sy := y
			if !topDown {
				sy = height - 1 - y
			}
			o := sy * row
			if o+row > len(src) {
				return nil, errf("DIB pixels are truncated")
			}
			for x := 0; x < width; x++ {
				i := o + x*4
				dst.SetNRGBA(x, y, rgba(src[i+2], src[i+1], src[i], src[i+3]))
			}
		}
	case 24:
		row := (width*3 + 3) &^ 3
		for y := 0; y < height; y++ {
			sy := y
			if !topDown {
				sy = height - 1 - y
			}
			o := sy * row
			if o+width*3 > len(src) {
				return nil, errf("DIB pixels are truncated")
			}
			for x := 0; x < width; x++ {
				i := o + x*3
				dst.SetNRGBA(x, y, rgba(src[i+2], src[i+1], src[i], 255))
			}
		}
	default:
		return nil, errf("unsupported DIB bit depth %d", bitCount)
	}
	return dst, nil
}

func rgba(r, g, b, a byte) color.NRGBA {
	if a == 0 {
		a = 255
	}
	return color.NRGBA{R: r, G: g, B: b, A: a}
}

func pngToDIB(pngBytes []byte) ([]byte, error) {
	img, err := png.Decode(bytes.NewReader(pngBytes))
	if err != nil {
		return nil, err
	}
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()
	nrgbaImg := image.NewNRGBA(image.Rect(0, 0, w, h))
	draw.Draw(nrgbaImg, nrgbaImg.Bounds(), img, b.Min, draw.Src)

	pixels := make([]byte, w*h*4)
	for y := 0; y < h; y++ {
		sy := h - 1 - y
		for x := 0; x < w; x++ {
			c := nrgbaImg.NRGBAAt(x, sy)
			o := (y*w + x) * 4
			a := c.A
			if a == 0 {
				a = 255
			}
			pixels[o+0] = c.B
			pixels[o+1] = c.G
			pixels[o+2] = c.R
			pixels[o+3] = a
		}
	}
	dib := make([]byte, 40+len(pixels))
	binary.LittleEndian.PutUint32(dib[0:4], 40)
	binary.LittleEndian.PutUint32(dib[4:8], uint32(w))
	binary.LittleEndian.PutUint32(dib[8:12], uint32(h))
	binary.LittleEndian.PutUint16(dib[12:14], 1)
	binary.LittleEndian.PutUint16(dib[14:16], 32)
	binary.LittleEndian.PutUint32(dib[20:24], uint32(len(pixels)))
	copy(dib[40:], pixels)
	return dib, nil
}

func imageToPNG(src []byte) ([]byte, int, int, error) {
	img, _, err := image.Decode(bytes.NewReader(src))
	if err != nil {
		return nil, 0, 0, err
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, 0, 0, err
	}
	r := img.Bounds()
	return buf.Bytes(), r.Dx(), r.Dy(), nil
}

func pngSize(pngBytes []byte) (int, int) {
	cfg, err := png.DecodeConfig(bytes.NewReader(pngBytes))
	if err != nil {
		return 0, 0
	}
	return cfg.Width, cfg.Height
}
