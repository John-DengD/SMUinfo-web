package httpx

import (
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type JWT struct {
	secret []byte
	expH   int
}

func NewJWT(secret string, expH int) JWT { return JWT{[]byte(secret), expH} }

func (j JWT) Generate(uid int64, role string) (string, error) {
	now := time.Now()
	t := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub":  strconv.FormatInt(uid, 10),
		"role": role,
		"iat":  now.Unix(),
		"exp":  now.Add(time.Duration(j.expH) * time.Hour).Unix(),
	})
	return t.SignedString(j.secret)
}

func (j JWT) Parse(tok string) (int64, string, error) {
	claims := jwt.MapClaims{}
	_, err := jwt.ParseWithClaims(tok, claims, func(t *jwt.Token) (any, error) { return j.secret, nil },
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}))
	if err != nil {
		return 0, "", err
	}
	sub, _ := claims["sub"].(string)
	uid, _ := strconv.ParseInt(sub, 10, 64)
	role, _ := claims["role"].(string)
	return uid, role, nil
}
