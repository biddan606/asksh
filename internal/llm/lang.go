package llm

import "regexp"

var hangulRE = regexp.MustCompile(`[\x{AC00}-\x{D7A3}]`)

// DetectLang returns "ko" if query contains Hangul syllables, otherwise "en".
func DetectLang(query string) string {
	if hangulRE.MatchString(query) {
		return "ko"
	}
	return "en"
}
