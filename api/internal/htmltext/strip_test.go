package htmltext

import "testing"

func TestStripToPlainText(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"empty", "", ""},
		{"whitespace only", "   \n\t  ", ""},
		{"plain text passthrough", "hello world", "hello world"},
		{"strip script tags but keep body (plain text sink is safe)",
			"<script>alert(1)</script>ok", "alert(1)ok"},
		{"br becomes newline", "a<br>b<br/>c<br />d", "a\nb\nc\nd"},
		{"paragraph block inserts double newline",
			"<p>one</p><p>two</p>", "one\n\ntwo"},
		{"list items prefixed with bullet",
			"<ul><li>a</li><li>b</li></ul>", ". a\n. b"},
		{"decode common entities",
			"&lt;a&gt;&amp;&nbsp;x&#39;y", "<a>& x'y"},
		{"collapse consecutive blank lines",
			"a\n\n\n\nb", "a\n\nb"},
		{"anchor tag stripped, content kept",
			`<a href="https://x">link</a>`, "link"},
		{"nested formatting stripped",
			"<p><strong>bold</strong> and <em>italic</em></p>", "bold and italic"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := StripToPlainText(tc.in)
			if got != tc.want {
				t.Fatalf("got %q, want %q", got, tc.want)
			}
		})
	}
}
