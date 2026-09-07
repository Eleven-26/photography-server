package jwtpkg

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// UserType 令牌主体类型：staff=员工（PC 后台/小程序后台/摄影师 App），customer=客户（H5/小程序客户端）。
// 旧令牌无此字段，解析后为空串——按 staff 处理以保持向后兼容。
const (
	UserTypeStaff    = "staff"
	UserTypeCustomer = "customer"
)

type Claims struct {
	UserID    int64  `json:"uid"`
	Username  string `json:"username"`
	CompanyID int64  `json:"cid"`
	StoreID   int64  `json:"sid"`
	RoleID    int64  `json:"rid"`
	UserType  string `json:"utype,omitempty"`
	jwt.RegisteredClaims
}

func Generate(secret string, issuer string, expireHours int, c Claims) (string, error) {
	c.RegisteredClaims = jwt.RegisteredClaims{
		Issuer:    issuer,
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Duration(expireHours) * time.Hour)),
		IssuedAt:  jwt.NewNumericDate(time.Now()),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, c)
	return token.SignedString([]byte(secret))
}

func Parse(secret, issuer, tokenString string) (*Claims, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(t *jwt.Token) (interface{}, error) {
		return []byte(secret), nil
	}, jwt.WithIssuer(issuer))
	if err != nil {
		return nil, err
	}
	if !token.Valid {
		return nil, errors.New("invalid token")
	}
	return claims, nil
}
