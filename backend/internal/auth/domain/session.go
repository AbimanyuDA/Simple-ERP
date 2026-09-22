package domain

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"time"
)

type PlainTokens struct {
	Access, Refresh, CSRF string
}

type Session struct {
	UserID, FamilyID                  string
	AccessHash, RefreshHash, CSRFHash string
	AccessExpiresAt                   time.Time
	RefreshIdleExpiresAt              time.Time
	RefreshAbsoluteExpiresAt          time.Time
}

func NewSessionTokens(userID, familyID string, pepper []byte, now time.Time) (PlainTokens, Session, error) {
	access, err := randomToken()
	if err != nil {
		return PlainTokens{}, Session{}, err
	}
	refresh, err := randomToken()
	if err != nil {
		return PlainTokens{}, Session{}, err
	}
	csrf, err := randomToken()
	if err != nil {
		return PlainTokens{}, Session{}, err
	}
	return PlainTokens{Access: access, Refresh: refresh, CSRF: csrf}, Session{
		UserID: userID, FamilyID: familyID,
		AccessHash: hashToken(access, pepper), RefreshHash: hashToken(refresh, pepper), CSRFHash: hashToken(csrf, pepper),
		AccessExpiresAt: now.Add(15 * time.Minute), RefreshIdleExpiresAt: now.Add(7 * 24 * time.Hour),
		RefreshAbsoluteExpiresAt: now.Add(30 * 24 * time.Hour),
	}, nil
}

func VerifyToken(token, expected string, pepper []byte) bool {
	return hmac.Equal([]byte(hashToken(token, pepper)), []byte(expected))
}

func hashToken(token string, pepper []byte) string {
	h := hmac.New(sha256.New, pepper)
	h.Write([]byte(token))
	return base64.RawURLEncoding.EncodeToString(h.Sum(nil))
}

func randomToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}
