package authservice

import (
	"os"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/twinj/uuid"
)

type AccessToken struct {
	UUID string
	Hash string
}

type RefreshToken struct {
	AccessUUID  string
	RefreshUUID string
	Hash        string
}

type Tokenizer interface {
	Generate(userID uint64) (*AccessToken, *RefreshToken, error)
}

type tokenizer struct{}

func NewTokenizer() Tokenizer {
	return &tokenizer{}
}

func (t *tokenizer) Generate(userID uint64) (*AccessToken, *RefreshToken, error) {
	access, err := generateAccessToken(userID)
	if err != nil {
		return nil, nil, err
	}

	refresh, err := generateRefreshToken(userID, access.UUID)
	if err != nil {
		return nil, nil, err
	}

	return access, refresh, nil
}

var (
	uuidV4 = uuid.NewV4
	uuidV5 = uuid.NewV5
)

func generateAccessToken(userID uint64) (*AccessToken, error) {
	id := uuidV4().String()
	expiry := time.Now().Add(time.Minute * 30).Unix()

	claims := jwt.MapClaims{
		"uuid":    id,
		"user_id": userID,
		"exp":     expiry,
	}

	t := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	hash, err := t.SignedString([]byte(os.Getenv("ACCESS_SECRET")))
	if err != nil {
		return nil, err
	}

	return &AccessToken{id, hash}, nil
}

func generateRefreshToken(userID uint64, accessUUID string) (*RefreshToken, error) {
	refreshUUID := uuidV5(uuid.NameSpaceURL, accessUUID).String()
	expiry := time.Now().Add(time.Hour * 24 * 7).Unix()

	claims := jwt.MapClaims{
		"access_uuid":  accessUUID,
		"refresh_uuid": refreshUUID,
		"user_id":      userID,
		"exp":          expiry,
	}

	t := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	hash, err := t.SignedString([]byte(os.Getenv("REFRESH_SECRET")))
	if err != nil {
		return nil, err
	}

	return &RefreshToken{accessUUID, refreshUUID, hash}, nil
}
