package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"golang.org/x/term"
)

const version = "0.1.0"

func usage() {
	fmt.Fprint(os.Stderr, `pb — pasteboard as a file

Usage:
  pb                 print clipboard to stdout
  ... | pb           copy stdin to clipboard
  pb -i FILE         copy FILE to clipboard
  pb -o FILE         paste clipboard to FILE
  pb --html          HTML flavor
  pb --files         Explorer file list
  pb --image         image as PNG
  pb --list          list available formats
  pb --clear         empty clipboard
  pb --json          JSON for --list / --files
  pb -0              NUL-separated file paths
  pb --lf            do not convert to CRLF on copy
  pb --crlf          keep CRLF on paste
  pb --raw           no newline conversion

  pb -h, --help
  pb -v, --version
`)
}

func main() {
	os.Exit(run(os.Args[1:]))
}

func run(args []string) int {
	fs := flag.NewFlagSet("pb", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	inFile := fs.String("i", "", "")
	outFile := fs.String("o", "", "")
	html := fs.Bool("html", false, "")
	files := fs.Bool("files", false, "")
	image := fs.Bool("image", false, "")
	text := fs.Bool("text", false, "")
	list := fs.Bool("list", false, "")
	clear := fs.Bool("clear", false, "")
	jsonOut := fs.Bool("json", false, "")
	zero := fs.Bool("0", false, "")
	lf := fs.Bool("lf", false, "")
	crlf := fs.Bool("crlf", false, "")
	raw := fs.Bool("raw", false, "")
	help := fs.Bool("h", false, "")
	helpLong := fs.Bool("help", false, "")
	ver := fs.Bool("v", false, "")
	verLong := fs.Bool("version", false, "")
	if err := fs.Parse(args); err != nil {
		fmt.Fprintln(os.Stderr, errf("%v", err))
		usage()
		return 2
	}
	if *help || *helpLong {
		usage()
		return 0
	}
	if *ver || *verLong {
		fmt.Println("pb " + version)
		return 0
	}

	flavorCount := 0
	flavor := FlavorAuto
	if *text {
		flavor = FlavorText
		flavorCount++
	}
	if *html {
		flavor = FlavorHTML
		flavorCount++
	}
	if *files {
		flavor = FlavorFiles
		flavorCount++
	}
	if *image {
		flavor = FlavorImage
		flavorCount++
	}
	if flavorCount > 1 {
		fmt.Fprintln(os.Stderr, errf("use only one of --text, --html, --files, --image"))
		return 2
	}

	switch {
	case *list:
		return cmdList(*jsonOut)
	case *clear:
		if err := clearClipboard(); err != nil {
			fail(err)
			return 1
		}
		return 0
	case *inFile != "":
		return cmdCopyFile(*inFile, flavor, *lf, *raw, *zero)
	case *outFile != "":
		return cmdPasteFile(*outFile, flavor, *crlf, *raw, *jsonOut, *zero)
	case !term.IsTerminal(int(os.Stdin.Fd())):
		return cmdCopyStdin(flavor, *lf, *raw, *zero)
	default:
		return cmdPasteStdout(flavor, *crlf, *raw, *jsonOut, *zero)
	}
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, err)
}

func cmdList(asJSON bool) int {
	names, err := listFormats()
	if err != nil {
		fail(err)
		return 1
	}
	if asJSON {
		enc, _ := json.Marshal(names)
		fmt.Println(string(enc))
		return 0
	}
	if len(names) == 0 {
		fmt.Fprintln(os.Stderr, errf("clipboard is empty"))
		return 1
	}
	fmt.Println(strings.Join(names, "\n"))
	return 0
}

func cmdCopyStdin(flavor Flavor, lf, raw, zero bool) int {
	b, err := io.ReadAll(os.Stdin)
	if err != nil {
		fail(errf("read stdin: %v", err))
		return 1
	}
	return copyBytes(b, flavor, lf, raw, zero)
}

func cmdCopyFile(path string, flavor Flavor, lf, raw, zero bool) int {
	if flavor == FlavorAuto {
		flavor = flavorFromExt(path)
	}
	b, err := os.ReadFile(path)
	if err != nil {
		fail(errf("%v", err))
		return 1
	}
	return copyBytes(b, flavor, lf, raw, zero)
}

func copyBytes(b []byte, flavor Flavor, lf, raw, zero bool) int {
	if flavor == FlavorAuto && sniffImage(b) {
		flavor = FlavorImage
	}
	switch flavor {
	case FlavorImage:
		pngBytes, _, _, err := imageToPNG(b)
		if err != nil {
			fail(errf("image: %v", err))
			return 1
		}
		if err := copyPNG(pngBytes); err != nil {
			fail(err)
			return 1
		}
	case FlavorHTML:
		s := decodeIncomingText(b)
		if err := copyHTML(s); err != nil {
			fail(err)
			return 1
		}
	case FlavorFiles:
		s := decodeIncomingText(b)
		paths := splitPaths(s, zero)
		if err := copyFiles(paths); err != nil {
			fail(err)
			return 1
		}
	default:
		s := decodeIncomingText(b)
		if !raw && !lf {
			s = toCRLF(s)
		}
		if err := copyText(s); err != nil {
			fail(err)
			return 1
		}
	}
	return 0
}

func cmdPasteFile(path string, flavor Flavor, crlf, raw, asJSON, zero bool) int {
	if flavor == FlavorAuto {
		flavor = flavorFromExt(path)
	}
	data, err := pasteBytes(flavor, crlf, raw, asJSON, zero, false)
	if err != nil {
		fail(err)
		return 1
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		fail(errf("%v", err))
		return 1
	}
	return 0
}

func cmdPasteStdout(flavor Flavor, crlf, raw, asJSON, zero bool) int {
	tty := term.IsTerminal(int(os.Stdout.Fd()))
	if flavor == FlavorAuto && tty {
		kind, err := clipboardKind()
		if err != nil {
			fail(err)
			return 1
		}
		if kind == FlavorImage {
			fmt.Fprint(os.Stderr, imageHint())
			return 2
		}
		flavor = kind
	}
	data, err := pasteBytes(flavor, crlf, raw, asJSON, zero, tty)
	if err != nil {
		fail(err)
		return 1
	}
	return writeOut(data, tty && flavor != FlavorImage)
}

func pasteBytes(flavor Flavor, crlf, raw, asJSON, zero, tty bool) ([]byte, error) {
	if flavor == FlavorAuto {
		kind, err := clipboardKind()
		if err != nil {
			return nil, err
		}
		flavor = kind
	}
	switch flavor {
	case FlavorImage:
		b, _, _, err := pastePNG()
		return b, err
	case FlavorHTML:
		s, err := pasteHTML()
		if err != nil {
			return nil, err
		}
		return []byte(s), nil
	case FlavorFiles:
		paths, err := pasteFiles()
		if err != nil {
			return nil, err
		}
		if asJSON {
			enc, err := json.Marshal(paths)
			if err != nil {
				return nil, err
			}
			return append(enc, '\n'), nil
		}
		s := joinPaths(paths, zero)
		if !zero && tty && !strings.HasSuffix(s, "\n") && s != "" {
			s += "\n"
		}
		return []byte(s), nil
	default:
		s, err := pasteText()
		if err != nil {
			return nil, err
		}
		if !raw && !crlf {
			s = toLF(s)
		}
		if tty && !strings.HasSuffix(s, "\n") && s != "" {
			s += "\n"
		}
		return []byte(s), nil
	}
}
