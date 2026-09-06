package identity

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"time"
	"github.com/golang-jwt/jwt/v5"
)

var ErrInvalidToken = errors.New("invalid token")

type Claims struct {
	SessionID string `json:"sid"`
	jwt.RegisteredClaims
}

type TokenManager struct { key []byte; accessTTL time.Duration }

func NewTokenManager(signingKey string, accessTTL time.Duration) *TokenManager {
	return &TokenManager{key: []byte(signingKey), accessTTL: accessTTL}
}

func (m *TokenManager) IssueAccessToken(userID, sessionID string) (string, time.Time, error) {
	now := time.Now().UTC()
	expiresAt := now.Add(m.accessTTL)
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, Claims{
		SessionID: sessionID,
		RegisteredClaims: jwt.RegisteredClaims{Subject:userID, IssuedAt:jwt.NewNumericDate(now), ExpiresAt:jwt.NewNumericDate(expiresAt)},
	})
	signed, err := token.SignedString(m.key)
	return signed, expiresAt, err
}

func (m *TokenManager) ParseAccessToken(raw string) (Claims, error) {
	var claims Claims
	token, err := jwt.ParseWithClaims(raw, &claims, func(token *jwt.Token) (any,error) {
		if token.Method.Alg() != jwt.SigningMethodHS256.Alg() { return nil, ErrInvalidToken }
		return m.key,nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}))
	if err != nil || !token.Valid || claims.Subject == "" || claims.SessionID == "" { return Claims{}, ErrInvalidToken }
	return claims,nil
}

func NewOpaqueToken(bytes int) (string,error) {
	raw:=make([]byte,bytes)
	if _,err:=rand.Read(raw); err!=nil { return "",err }
	return base64.RawURLEncoding.EncodeToString(raw),nil
}

func HashOpaqueToken(raw string) string {
	sum:=sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}
