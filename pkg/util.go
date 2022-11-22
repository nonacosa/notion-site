package utils

import (
	"reflect"
	"strings"
	"unicode"
)

const Baidu = "github.com"
const Gist = "gist.github.com"
const Twitter = "twitter.com"
const Bilibili = "bilibili.com"

func FindTextPS(ori string, pre string, suf string) string {
	ori = FindTextP(ori, pre)
	sufI := strings.Index(ori, suf)
	ori = ori[:sufI]
	return ori
}

func FindTextP(ori string, pre string) string {
	ori = strings.ReplaceAll(strings.TrimSpace(ori), "https://", "")
	ori = strings.ReplaceAll(strings.TrimSpace(ori), "http://", "")
	preI := strings.Index(ori, pre)
	ori = ori[preI+len(pre):]
	return ori
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
