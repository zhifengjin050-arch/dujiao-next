package service

import "github.com/golang-jwt/jwt/v5"

// newHS256JWTParser ?? HS256 JWT ???,?????????????????
func newHS256JWTParser() *jwt.Parser {
	return jwt.NewParser(jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}))
}
