package auth

import (
	"crypto/rand"
	"encoding/base64"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var secret = func() []byte {
	if s := os.Getenv("JWT_SECRET"); s != "" {
		return []byte(s)
	}
	b := make([]byte, 32)
	rand.Read(b)
	return []byte(base64.StdEncoding.EncodeToString(b))
}()

const tokenTTL = 20 * time.Minute

type Claims struct {
	UID  int    `json:"uid"`
	Role string `json:"role"`
	jwt.RegisteredClaims
}

func Sign(uid int, role string) (string, error) {
	claims := &Claims{
		UID:  uid,
		Role: role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(tokenTTL)),
		},
	}
	tk := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return tk.SignedString(secret)
}

func Parse(tkn string) (*Claims, error) {
	p, err := jwt.ParseWithClaims(
		tkn, &Claims{},
		func(*jwt.Token) (any, error) { return secret, nil },
	)
	if err != nil {
		return nil, err
	}
	return p.Claims.(*Claims), nil
}
