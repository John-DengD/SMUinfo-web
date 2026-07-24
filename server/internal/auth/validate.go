package auth

import (
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/John-DengD/smu-deal/server/internal/httpx"
)

var (
	studentNoRe = regexp.MustCompile(`^\d{12}$`)
	nameRe      = regexp.MustCompile(`^[\p{Han}A-Za-z·.\- ]+$`)
	wsRe        = regexp.MustCompile(`\s+`)
)

var obviousFakeStudentNos = []string{
	"000000000000",
	"111111111111",
	"123456789012",
}

func validateStudentNo(value string) (string, error) {
	studentNo := strings.TrimSpace(value)
	if !studentNoRe.MatchString(studentNo) {
		return "", httpx.Biz("学号必须是 12 位纯数字")
	}
	for _, fake := range obviousFakeStudentNos {
		if fake == studentNo {
			return "", httpx.Biz("请填写真实学号")
		}
	}
	return studentNo, nil
}

func validateName(value string) (string, error) {
	name := wsRe.ReplaceAllString(strings.TrimSpace(value), " ")
	n := utf8.RuneCountInString(name)
	if n < 2 || n > 20 {
		return "", httpx.Biz("姓名长度需为 2-20 个字符")
	}
	if !nameRe.MatchString(name) {
		return "", httpx.Biz("姓名只能包含中文、英文字母、空格、点号、中点或连字符，不能包含数字")
	}
	return name, nil
}
