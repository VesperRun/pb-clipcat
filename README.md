# pb-clipcat

Command: **`pb`**. Pasteboard as a file. Clipboard is stdin and stdout.

```text
echo hello | pb
pb
pb -o shot.png
```

Windows first. One binary. No account.

```text
git clone https://github.com/VesperRun/pb-clipcat.git
```

## Install

```powershell
git clone https://github.com/VesperRun/pb-clipcat.git
cd pb-clipcat
go build -ldflags="-s -w" -o pb.exe .
```

Put `pb.exe` on your PATH.

## Use

```text
pb                 print clipboard (text, or file list)
... | pb           copy stdin to clipboard
pb -i FILE         copy a file
pb -o FILE         paste to a file
pb --html          HTML flavor
pb --files         Explorer file list (CF_HDROP)
pb --image         image as PNG
pb --list          formats on the clipboard
pb --clear         empty the clipboard
```

Copy a table from Excel, then:

```powershell
pb > out.csv
```

Win+Shift+S, then:

```powershell
pb -o shot.png
```

PowerShell 5 mangles binary `>`. For images always use `-o`.

HTML from a browser:

```powershell
pb --html -o page.html
```

Files copied in Explorer:

```powershell
pb --files
```

## Flavors

If you do not pass a flavor, `pb` does this:

- Terminal: print text, or file paths. If the clipboard is only an image, it tells you to use `pb -o shot.png`.
- Redirected stdout with no text: write PNG (so `pb > shot.png` works in cmd).
- `-o shot.png` / `-i photo.jpg`: inferred from the extension.

Copy converts `\n` to CRLF so Notepad is happy. Paste converts CRLF to `\n` so pipes are happy. `--raw` skips that.

## Not

This is not Apple's `pbcopy` / `pbpaste`. It is not the Clipboard Project (`cb`). It is a brick: one clipboard, one process, then you leave.
