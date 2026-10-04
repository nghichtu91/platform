package util

import (
	"bytes"
	"regexp"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

// 下面这个也是stackoverflow抄的
var matchFirstCap = regexp.MustCompile("(.)([A-Z][a-z]+)")
var matchAllCap = regexp.MustCompile("([a-z0-9])([A-Z])")

// ToSnakeCase 驼峰转下划线
func ToSnakeCase(str string) string {
	snake := matchFirstCap.ReplaceAllString(str, "${1}_${2}")
	snake = matchAllCap.ReplaceAllString(snake, "${1}_${2}")
	return strings.ToLower(snake)
}

// ToUpperCamelCase 转大写驼峰
func ToUpperCamelCase(s string) string {
	return ToCamelCase(s, true)
}

// ToLowerCamelCase 转小写驼峰
func ToLowerCamelCase(s string) string {
	return ToCamelCase(s, false)
}

// 下面这些是从golint里抄的

// ToCamelCase returns a different name if it should be different.
// 用这个函数产生的struct定义基本符合golint要求
func ToCamelCase(name string, needUpper bool) (should string) {
	buffer := new(bytes.Buffer)

	// Split camelCase at any lower->upper transition, and split on underscores.
	// Check each word for common initialisms.
	runes := []rune(strings.TrimSpace(name))
	w, i := 0, 0 // index of start of word, scan
	for i+1 <= len(runes) {
		eow := false // whether we hit the end of a word
		if i+1 == len(runes) {
			eow = true
		} else if !(unicode.IsDigit(runes[i+1]) || unicode.IsLetter(runes[i+1])) {
			// underscore; shift the remainder forward over any run of underscores
			eow = true
			n := 1
			for i+n+1 < len(runes) && !(unicode.IsDigit(runes[i+n+1]) || unicode.IsLetter(runes[i+n+1])) {
				n++
			}

			copy(runes[i+1:], runes[i+n+1:])
			runes = runes[:len(runes)-n]
		} else if unicode.IsLower(runes[i]) && !unicode.IsLower(runes[i+1]) {
			// lower->non-lower
			eow = true
		} else if !unicode.IsLetter(runes[i]) && unicode.IsLetter(runes[i+1]) {
			// non-letter->letter
			eow = true
		}
		i++
		if !eow {
			continue
		}

		// [w,i) is a word.
		word := string(runes[w:i])
		if u := strings.ToUpper(word); commonInitialisms[u] {
			// Keep consistent case, which is lowercase only at the start.
			if w == 0 && unicode.IsLower(runes[w]) {
				u = strings.ToLower(u)
			}
			// All the common initialisms are ASCII,
			// so we can replace the bytes exactly.
			copy(runes[w:], []rune(u))
		} else if w > 0 && strings.ToLower(word) == word {
			// already all lowercase, and not the first word, so uppercase the first character.
			runes[w] = unicode.ToUpper(runes[w])
		}

		// trim non-letters from beginning
		if w == 0 {
			if needUpper {
				runes[0] = unicode.ToUpper(runes[0])
			} else {
				runes[0] = unicode.ToLower(runes[0])
			}
		}

		buffer.WriteString(string(runes[w:i]))

		w = i
	}

	return buffer.String()
}

// commonInitialisms is a set of common initialisms.
// Only add entries that are highly unlikely to be non-initialisms.
// For instance, "ID" is fine (Freudian code is rare), but "AND" is not.
var commonInitialisms = map[string]bool{
	"ACL":   true,
	"API":   true,
	"ASCII": true,
	"CPU":   true,
	"CSS":   true,
	"DNS":   true,
	"EOF":   true,
	"GUID":  true,
	"HTML":  true,
	"HTTP":  true,
	"HTTPS": true,
	"ID":    true,
	"IP":    true,
	"JSON":  true,
	"LHS":   true,
	"QPS":   true,
	"RAM":   true,
	"RHS":   true,
	"RPC":   true,
	"SLA":   true,
	"SMTP":  true,
	"SQL":   true,
	"SSH":   true,
	"TCP":   true,
	"TLS":   true,
	"TTL":   true,
	"UDP":   true,
	"UI":    true,
	"UID":   true,
	"UUID":  true,
	"URI":   true,
	"URL":   true,
	"UTF8":  true,
	"VM":    true,
	"XML":   true,
	"XMPP":  true,
	"XSRF":  true,
	"XSS":   true,
}

// SimpleGetUintFromString 从字符串中获取第一个正整数并转换为int返回
// 小数点，负号等等全部忽略不计
// 支持unicode
func SimpleGetUintFromString(in string) int {
	var (
		st, et  int
		matched bool
	)

	for idx, r := range in {
		if unicode.IsDigit(r) {
			if !matched {
				st = idx
				et = idx + 1
				matched = true
			} else {
				et++
			}
		} else {
			if matched {
				break
			}
		}
	}

	if et == 0 {
		return 0
	}

	num, _ := strconv.Atoi(in[st:et])
	return num
}

func CalcGameStringLen(str string) int {
	var total int
	if len(str) == 0 {
		return total
	}
	for _, char := range str {
		// 默认: 非ASCII字符统一按照长度2计算
		if utf8.RuneLen(char) > 1 {
			total += 2
		} else {
			total += 1
		}
	}
	return total
}
