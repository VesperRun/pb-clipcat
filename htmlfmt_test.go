package main

import "testing"

func TestWrapUnwrapHTML(t *testing.T) {
	got := wrapHTML("<b>hi</b>")
	if want := "StartHTML:"; !contains(got, want) {
		t.Fatalf("missing %s in %q", want, got)
	}
	inner := unwrapHTML(got)
	if inner != "<b>hi</b>" && inner != "<!--StartFragment--><b>hi</b><!--EndFragment-->" {
		if unwrapHTML(got) == "" {
			t.Fatalf("unwrap empty")
		}
	}
	frag := unwrapHTML(got)
	if !contains(frag, "<b>hi</b>") {
		t.Fatalf("fragment %q", frag)
	}
}

func TestStripTags(t *testing.T) {
	if got := stripTags("<b>hi &amp; bye</b>"); got != "hi & bye" {
		t.Fatalf("got %q", got)
	}
}

func TestHTMLOffsetZero(t *testing.T) {
	n, ok := htmlOffset("StartHTML:0000000000\r\n", "StartHTML:")
	if !ok || n != 0 {
		t.Fatalf("n=%d ok=%v", n, ok)
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(sub) == 0 ||
		func() bool {
			for i := 0; i+len(sub) <= len(s); i++ {
				if s[i:i+len(sub)] == sub {
					return true
				}
			}
			return false
		}())
}
