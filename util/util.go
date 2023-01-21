package util

import (
	"math/rand"
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

const (
	ERR_NOT_AUTHORIZED = "ERR_NOT_AUTHORIZED"
)

type Photo struct {
	gorm.Model
	Name   string `json:"name"`
	UserID uint   `json:"user_id"`
	CID    string `json:"cid" gorm:"column:cid"`
}

type User struct {
	gorm.Model
	Handle string  `json:"handle" gorm:"unique"`
	ApiKey string  `json:"apikey" gorm:"column:apikey"`
	Photos []Photo `json:photos`
}

func GenerateToken(n int) string {
	var letters = []rune("ABCDEFGHIJKLMNOPQRSTUVWXYZ1234567890")

	b := make([]rune, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}

func WithUser(f func(echo.Context, *User) error) func(echo.Context) error {
	return func(c echo.Context) error {
		u, ok := c.Get("user").(*User)
		if !ok {
			return &HttpError{
				Code:    http.StatusUnauthorized,
				Reason:  "invalid API key",
				Details: "endpoint not called with proper authentication",
			}
		}
		return f(c, u)
	}
}

type HttpError struct {
	Code    int    `json:"code,omitempty"`
	Reason  string `json:"reason"`
	Details string `json:"details"`
}

func (he HttpError) Error() string {
	if he.Details == "" {
		return he.Reason
	}
	return he.Reason + ": " + he.Details
}

func ExtractAuth(c echo.Context) (string, error) {
	auth := c.Request().Header.Get("Authentication")
	//	undefined will be the auth value if ESTUARY_TOKEN cookie is removed.
	if auth == "" || auth == "undefined" {
		return "", &HttpError{
			Code:   http.StatusUnauthorized,
			Reason: "Authentication Missing",
		}
	}

	parts := strings.Split(auth, " ")

	if parts[0] != "Bearer" {
		return "", &HttpError{
			Code:   http.StatusUnauthorized,
			Reason: "Authentication Missing 'Bearer'",
		}
	}

	if len(parts) != 2 {
		return "", &HttpError{
			Code:   http.StatusUnauthorized,
			Reason: "Invalid API key",
		}
	}
	return parts[1], nil
}

func CreateDatabase() (*gorm.DB, error) {
	db, err := gorm.Open(sqlite.Open("estuphotos.db"), &gorm.Config{})
	if err != nil {
		return nil, err
	}
	db.AutoMigrate(&Photo{})
	db.AutoMigrate(&User{})

	return db, nil
}
