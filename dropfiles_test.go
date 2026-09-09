package main

import (
	"path/filepath"
	"testing"
)

func TestHDROPRoundTrip(t *testing.T) {
	in := []string{`C:\Users\jaeso\a.txt`, `C:\Temp\b.png`}
	raw := serializeHDROP(in)
	out, err := parseHDROP(raw)
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 2 || out[0] != in[0] || out[1] != in[1] {
		t.Fatalf("got %#v", out)
	}
}

func TestFlavorFromExt(t *testing.T) {
	if flavorFromExt("shot.PNG") != FlavorImage {
		t.Fatal("png")
	}
	if flavorFromExt("page.HTML") != FlavorHTML {
		t.Fatal("html")
	}
	if flavorFromExt(filepath.Join("x", "n.txt")) != FlavorText {
		t.Fatal("txt")
	}
}

func TestSplitJoinPaths(t *testing.T) {
	s := "a.txt\nb.txt\n"
	p := splitPaths(s, false)
	if len(p) != 2 {
		t.Fatalf("%v", p)
	}
	z := splitPaths("a.txt\x00b.txt\x00", true)
	if len(z) != 2 {
		t.Fatalf("%v", z)
	}
}
