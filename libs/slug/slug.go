package slug

import (
	"regexp"
	"strings"
	"time"
)

var transliterations = map[string]string{
	"а": "a", "б": "b", "в": "v", "г": "g", "д": "d",
	"е": "e", "ё": "yo", "ж": "zh", "з": "z", "и": "i",
	"й": "y", "к": "k", "л": "l", "м": "m", "н": "n",
	"о": "o", "п": "p", "р": "r", "с": "s", "т": "t",
	"у": "u", "ф": "f", "х": "h", "ц": "ts", "ч": "ch",
	"ш": "sh", "щ": "sch", "ъ": "", "ы": "y", "ь": "",
	"э": "e", "ю": "yu", "я": "ya",
	"А": "a", "Б": "b", "В": "v", "Г": "g", "Д": "d",
	"Е": "e", "Ё": "yo", "Ж": "zh", "З": "z", "И": "i",
	"Й": "y", "К": "k", "Л": "l", "М": "m", "Н": "n",
	"О": "o", "П": "p", "Р": "r", "С": "s", "Т": "t",
	"У": "u", "Ф": "f", "Х": "h", "Ц": "ts", "Ч": "ch",
	"Ш": "sh", "Щ": "sch", "Ъ": "", "Ы": "y", "Ь": "",
	"Э": "e", "Ю": "yu", "Я": "ya",
}

func transliterate(text string) string {
	for cyrillic, latin := range transliterations {
		text = strings.ReplaceAll(text, cyrillic, latin)
	}
	return text
}

func GenerateSlug(text string, t time.Time) string {
	// Transliterate Russian characters to Latin
	text = transliterate(text)

	// Take first 100 chars to create meaningful slug
	if len(text) > 100 {
		text = text[:100]
	}

	// Convert to lowercase
	text = strings.ToLower(text)

	// Replace special characters
	reg := regexp.MustCompile("[^a-z0-9]+")
	text = reg.ReplaceAllString(text, "-")

	// Remove leading/trailing hyphens
	text = strings.Trim(text, "-")

	if len(text) > 90 {
		text = text[:90]
	}

	// Append timestamp
	text = text + "-" + t.Format("2006-01-02")

	return text
}
