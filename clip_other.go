//go:build !windows

package main

func listFormats() ([]string, error) { return nil, unsupported() }
func clearClipboard() error          { return unsupported() }
func pasteText() (string, error)     { return "", unsupported() }
func pasteHTML() (string, error)     { return "", unsupported() }
func pasteFiles() ([]string, error)  { return nil, unsupported() }
func pastePNG() ([]byte, int, int, error) {
	return nil, 0, 0, unsupported()
}
func clipboardKind() (Flavor, error) { return 0, unsupported() }
func copyText(string) error          { return unsupported() }
func copyHTML(string) error          { return unsupported() }
func copyFiles([]string) error       { return unsupported() }
func copyPNG([]byte) error           { return unsupported() }
func imageHint() string              { return "pb: clipboard holds an image\n" }

func unsupported() error {
	return errf("only Windows is supported in this version")
}
