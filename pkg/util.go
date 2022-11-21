package utils

import (
	"strings"
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
