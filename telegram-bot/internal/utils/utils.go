package utils

import (
	"os"
	"regexp"
	"strings"
)

func GetEnv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

func MustInt64(s string) int64 {
	var n int64
	var sign int64 = 1
	i := 0
	if len(s) > 0 && s[0] == '-' {
		sign = -1
		i++
	}
	for ; i < len(s); i++ {
		c := s[i]
		if c < '0' || c > '9' {
			break
		}
		n = n*10 + int64(c-'0')
	}
	return n * sign
}

func Itoa64(v int64) string {
	if v == 0 {
		return "0"
	}
	neg := v < 0
	if neg {
		v = -v
	}
	var b [32]byte
	i := len(b)
	for v > 0 {
		i--
		b[i] = byte('0' + v%10)
		v /= 10
	}
	if neg {
		i--
		b[i] = '-'
	}
	return string(b[i:])
}

func Mdv2Escape(s string) string {
	replacer := strings.NewReplacer(
		`_`, `\_`,
		`*`, `\*`,
		`[`, `\[`,
		`]`, `\]`,
		`(`, `\(`,
		`)`, `\)`,
		`~`, `\~`,
		"`", "\\`",
		`>`, `\>`,
		`#`, `\#`,
		`+`, `\+`,
		`-`, `\-`,
		`=`, `\=`,
		`|`, `\|`,
		`{`, `\{`,
		`}`, `\}`,
		`.`, `\.`,
		`!`, `\!`,
	)
	return replacer.Replace(s)
}

// NormalizeButtonText убирает эмодзи и нормализует текст для сравнения
func NormalizeButtonText(text string) string {
	// Убираем все эмодзи (Unicode range для эмодзи)
	emojiRegex := regexp.MustCompile(`[\x{1F300}-\x{1F9FF}]|[\x{2600}-\x{26FF}]|[\x{2700}-\x{27BF}]|[\x{1F600}-\x{1F64F}]|[\x{1F680}-\x{1F6FF}]|[\x{1F1E0}-\x{1F1FF}]`)
	normalized := emojiRegex.ReplaceAllString(text, "")
	// Убираем лишние пробелы и приводим к нижнему регистру
	normalized = strings.ToLower(strings.TrimSpace(strings.ReplaceAll(normalized, "  ", " ")))
	return normalized
}
