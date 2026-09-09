package main

import (
	"fmt"
	"strconv"
	"strings"
)

const htmlHeaderTmpl = "Version:0.9\r\nStartHTML:%010d\r\nEndHTML:%010d\r\nStartFragment:%010d\r\nEndFragment:%010d\r\n"

func wrapHTML(fragment string) string {
	body := fragment
	if !strings.Contains(strings.ToLower(fragment), "<!--startfragment-->") {
		body = "<!--StartFragment-->" + fragment + "<!--EndFragment-->"
	}
	if !strings.Contains(strings.ToLower(body), "<html") {
		body = "<html>\r\n<body>\r\n" + body + "\r\n</body>\r\n</html>"
	}
	header := fmt.Sprintf(htmlHeaderTmpl, 0, 0, 0, 0)
	startHTML := len(header)
	startFrag := startHTML + strings.Index(strings.ToLower(body), "<!--startfragment-->")
	if i := strings.Index(strings.ToLower(body), "<!--startfragment-->"); i >= 0 {
		startFrag = startHTML + i + len("<!--StartFragment-->")
	} else {
		startFrag = startHTML
	}
	endFrag := startHTML + len(body)
	if i := strings.Index(strings.ToLower(body), "<!--endfragment-->"); i >= 0 {
		endFrag = startHTML + i
	}
	endHTML := startHTML + len(body)
	header = fmt.Sprintf(htmlHeaderTmpl, startHTML, endHTML, startFrag, endFrag)
	return header + body
}

func unwrapHTML(data string) string {
	start, ok1 := htmlOffset(data, "StartFragment:")
	end, ok2 := htmlOffset(data, "EndFragment:")
	if ok1 && ok2 && start >= 0 && end <= len(data) && start < end {
		return data[start:end]
	}
	start, ok1 = htmlOffset(data, "StartHTML:")
	end, ok2 = htmlOffset(data, "EndHTML:")
	if ok1 && ok2 && start >= 0 && end <= len(data) && start < end {
		return data[start:end]
	}
	return data
}

func htmlOffset(data, key string) (int, bool) {
	i := strings.Index(data, key)
	if i < 0 {
		return 0, false
	}
	i += len(key)
	j := i
	for j < len(data) && data[j] >= '0' && data[j] <= '9' {
		j++
	}
	n, err := strconv.Atoi(strings.TrimLeft(data[i:j], "0"))
	if err != nil {
		if data[i:j] == "0000000000" || data[i:j] == "0" {
			return 0, true
		}
		return 0, false
	}
	return n, true
}

func stripTags(html string) string {
	var b strings.Builder
	inTag := false
	for i := 0; i < len(html); i++ {
		c := html[i]
		switch {
		case c == '<':
			inTag = true
		case c == '>':
			inTag = false
		case !inTag:
			b.WriteByte(c)
		}
	}
	s := strings.ReplaceAll(b.String(), "&nbsp;", " ")
	s = strings.ReplaceAll(s, "&amp;", "&")
	s = strings.ReplaceAll(s, "&lt;", "<")
	s = strings.ReplaceAll(s, "&gt;", ">")
	s = strings.ReplaceAll(s, "&quot;", "\"")
	return strings.TrimSpace(s)
}
