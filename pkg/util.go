package pkg

import (
	"errors"
	"github.com/dlclark/regexp2"
	"path"
	"reflect"
	"strings"
	"unicode"
)

const Gist = "gist.github.com"
const Twitter = "twitter.com"
const X = "x.com"
const Jsfiddle = "jsfiddle.net"
const Bilibili = "bilibili.com"
const RegexBili = `((?<=\.com\/video\/).*(?=\/))|((?<=bvid=).*(?=&cid?))`
const RegexYoutube = `(?<=\.com\/watch\?v=).*`
// match status id (digits) after /status/ until end, slash or question
const RegexTwitterId = `(?<=status\/)[^\/\?]+`
// match username between domain and /status
const RegexTwitterUser = `(?<=com\/)[^\/]+(?=\/status)`
const RegexJsfiddle = `(?<=jsfiddle\.net\/).*(?=\/)`

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

func HasEmpty(s string) bool {
	if s == "" || len(s) == 0 {
		return true
	}
	return false
}

func RemoveSuffix(str string) (string, error) {
	if HasEmpty(str) {
		return "", errors.New("Parameter  is an empty string")
	}
	filenameWithSuffix := path.Base(str)
	fileSuffix := path.Ext(filenameWithSuffix)
	filenameOnly := strings.TrimSuffix(filenameWithSuffix, fileSuffix)
	return filenameOnly, nil
}
