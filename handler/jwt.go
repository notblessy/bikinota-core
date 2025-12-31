package handler

import (
	"errors"
	"os"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/labstack/echo/v4"
)

type jwtClaims struct {
	ID    uint   `json:"id"`
	Email string `json:"email"`
	Name  string `json:"name"`
	jwt.RegisteredClaims
}

func signJWTToken(id uint, email, name string) (string, error) {
	claims := &jwtClaims{
		ID:    id,
		Email: email,
		Name:  name,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour * time.Duration(24*7))),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	t, err := token.SignedString([]byte(os.Getenv("JWT_SECRET")))
	if err != nil {
		return "", err
	}

	return t, nil
}

func validateToken(tokenString string) (jwtClaims, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}

		return []byte(os.Getenv("JWT_SECRET")), nil
	})

	if err != nil || !token.Valid {
		return jwtClaims{}, err
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || !token.Valid {
		return jwtClaims{}, errors.New("invalid token claims")
	}

	id, ok := claims["id"].(float64)
	if !ok {
		return jwtClaims{}, errors.New("user id not found in claims")
	}

	email, ok := claims["email"].(string)
	if !ok {
		return jwtClaims{}, errors.New("email not found in claims")
	}

	name, ok := claims["name"].(string)
	if !ok {
		return jwtClaims{}, errors.New("name not found in claims")
	}

	return jwtClaims{
		ID:    uint(id),
		Email: email,
		Name:  name,
	}, nil
}

func authSession(c echo.Context) (jwtClaims, error) {
	u := c.Get("user")
	if u == nil {
		return jwtClaims{}, errors.New("missing session")
	}

	user, ok := u.(jwtClaims)
	if !ok {
		return jwtClaims{}, errors.New("invalid session")
	}

	return user, nil
}

type JWTMiddleware struct{}

func NewJWTMiddleware() *JWTMiddleware {
	return &JWTMiddleware{}
}

func (m *JWTMiddleware) ValidateJWT(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		authHeader := c.Request().Header.Get("Authorization")
		if authHeader == "" {
			return c.JSON(401, response{
				Success: false,
				Message: "authorization token is required",
			})
		}

		token := strings.TrimPrefix(authHeader, "Bearer ")
		if token == authHeader {
			return c.JSON(401, response{
				Success: false,
				Message: "token is malformed",
			})
		}

		user, err := validateToken(token)
		if err != nil || user.ID == 0 {
			return c.JSON(401, response{
				Success: false,
				Message: "cannot validate token: " + err.Error(),
			})
		}

		c.Set("user", user)

		return next(c)
	}
}

