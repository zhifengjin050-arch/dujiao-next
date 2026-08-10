// Package htmltext ????? HTML ??????????
// ???????Telegram ??????? HTML ???,????/??????????
//
// ??:telegram-bot ???(?? go.mod)???????,???????
package htmltext

import "strings"

// StripToPlainText ? HTML ????????:????/??/????,???????
// ????????????,??????,???????
func StripToPlainText(raw string) string {
	s := strings.TrimSpace(raw)
	if s == "" {
		return ""
	}
	replacer := strings.NewReplacer(
		"<br>", "\n", "<br/>", "\n", "<br />", "\n",
		"</p>", "\n\n", "</div>", "\n", "</li>", "\n",
		"<li>", ". ",
	)
	s = replacer.Replace(s)

	var out strings.Builder
	out.Grow(len(s))
	inTag := false
	for _, r := range s {
		switch {
		case r == '<':
			inTag = true
		case r == '>':
			inTag = false
		case !inTag:
			out.WriteRune(r)
		}
	}
	text := out.String()

	entityReplacer := strings.NewReplacer(
		"&nbsp;", " ", "&amp;", "&", "&lt;", "<", "&gt;", ">",
		"&quot;", "\"", "&#39;", "'",
	)
	text = entityReplacer.Replace(text)

	lines := strings.Split(text, "\n")
	cleaned := make([]string, 0, len(lines))
	prevBlank := false
	for _, line := range lines {
		trimmed := strings.TrimRight(line, " \t")
		if trimmed == "" {
			if prevBlank {
				continue
			}
			prevBlank = true
		} else {
			prevBlank = false
		}
		cleaned = append(cleaned, trimmed)
	}
	return strings.TrimSpace(strings.Join(cleaned, "\n"))
}
