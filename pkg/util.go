package utils

import (
	"github.com/dlclark/regexp2"
	"reflect"
	"strings"
	"unicode"
)

const Gist = "gist.github.com"
const Twitter = "twitter.com"
const Bilibili = "bilibili.com"
const RegexBili = `((?<=\.com\/video\/).*(?=\/))|((?<=bvid=).*(?=&cid?))`
const RegexYoutube = `(?<=\.com\/watch\?v=).*`
const RegexTwitterId = `(?<=status\/).*(?=\?)`
const RegexTwitterUser = `(?<=com\/).*(?=\/status)`

func FindTextP(ori string, pre string) string {
	ori = strings.ReplaceAll(strings.TrimSpace(ori), "https://", "")
	ori = strings.ReplaceAll(strings.TrimSpace(ori), "http://", "")
	preI := strings.Index(ori, pre)
	ori = ori[preI+len(pre):]
	return ori
}

func FindUrlContext(regex string, url string) string {
	var res string
	reg, _ := regexp2.Compile(regex, 0)
	m, _ := reg.FindStringMatch(url)
	if m != nil {
		res = m.String()
	}
	return res
}

func CamelCaseToUnderscore(s string) string {
	var output []rune
	for i, r := range s {
		if i == 0 {
			output = append(output, unicode.ToLower(r))
			continue
		}
		if unicode.IsUpper(r) {
			output = append(output, '_')
		}
		if unicode.IsNumber(r) {
			output = append(output, '_')
		}
		output = append(output, unicode.ToLower(r))
	}
	return string(output)
}

func GetBlockType(block any) string {
	blockType := strings.Replace(reflect.TypeOf(block).String(), "*notion.", "", -1)
	return CamelCaseToUnderscore(strings.ReplaceAll(blockType, "Block", ""))
}

func Filter[T any](s []T, cond func(t T) bool) []T {
	var res []T
	for _, v := range s {
		if cond(v) {
			res = append(res, v)
		}
	}
	return res
}
