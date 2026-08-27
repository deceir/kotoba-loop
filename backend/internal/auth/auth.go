package auth

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

func Hash(password string) (string, error) {
	if len(password) < 8 {
		return "", errors.New("password must be at least 8 characters")
	}
	b, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(b), err
}
func Verify(hash, password string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}
func Sign(userID int64, secret string) (string, error) {
	c := jwt.MapClaims{"sub": userID, "exp": time.Now().Add(7 * 24 * time.Hour).Unix()}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, c).SignedString([]byte(secret))
}
func Parse(token, secret string) (int64, error) {
	t, err := jwt.Parse(token, func(t *jwt.Token) (any, error) {
		if t.Method != jwt.SigningMethodHS256 {
			return nil, errors.New("invalid signing method")
		}
		return []byte(secret), nil
	})
	if err != nil || !t.Valid {
		return 0, errors.New("invalid token")
	}
	sub, ok := t.Claims.(jwt.MapClaims)["sub"].(float64)
	if !ok {
		return 0, errors.New("invalid subject")
	}
	return int64(sub), nil
}
