package audit

import "testing"

func TestParseUserAgent(t *testing.T) {
	cases := []struct {
		ua      string
		device  string
		browser string
		os      string
	}{
		{
			"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0 Safari/537.36",
			"Desktop", "Chrome", "Windows",
		},
		{
			"Mozilla/5.0 (iPhone; CPU iPhone OS 17_0 like Mac OS X) AppleWebKit/605.1.15 Mobile/15E148 Safari/604.1",
			"Mobile", "Safari", "iOS",
		},
		{
			"Mozilla/5.0 (Linux; Android 14; Pixel 8) AppleWebKit/537.36 Chrome/120.0 Mobile Safari/537.36",
			"Mobile", "Chrome", "Android",
		},
		{"curl/8.4.0", "Desktop", "curl", "Other"},
		{"", "unknown", "unknown", "unknown"},
	}
	for _, tc := range cases {
		device, browser, os := ParseUserAgent(tc.ua)
		if device != tc.device || browser != tc.browser || os != tc.os {
			t.Errorf("ParseUserAgent(%q) = (%s,%s,%s) want (%s,%s,%s)", tc.ua, device, browser, os, tc.device, tc.browser, tc.os)
		}
	}
}
