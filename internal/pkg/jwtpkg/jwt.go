package jwtpkg

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"strconv"
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
	now := time.Now()
	jti, err := newJTI()
	if err != nil {
		return "", err
	}
	c.RegisteredClaims = jwt.RegisteredClaims{
		Issuer:    issuer,
		Subject:   strconv.FormatInt(c.UserID, 10), // 主体=用户ID，便于服务端识别
		ID:        jti,                             // jti：唯一令牌 ID，登出/吊销时入黑名单
		ExpiresAt: jwt.NewNumericDate(now.Add(time.Duration(expireHours) * time.Hour)),
		IssuedAt:  jwt.NewNumericDate(now),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, c)
	return token.SignedString([]byte(secret))
}

// Parse 解析并校验 JWT。
// WithValidMethods 显式锁定 HS256，彻底排除"算法混淆"攻击面（攻击者换 alg 头
// 诱导服务端用对称密钥验签的经典漏洞）。
func Parse(secret, issuer, tokenString string) (*Claims, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(t *jwt.Token) (interface{}, error) {
		return []byte(secret), nil
	}, jwt.WithIssuer(issuer), jwt.WithValidMethods([]string{"HS256"}))
	if err != nil {
		return nil, err
	}
	if !token.Valid {
		return nil, errors.New("invalid token")
	}
	return claims, nil
}

// BlacklistKey 登出/吊销黑名单键（value=userID，TTL=令牌剩余有效期）。
// 旧令牌（签发时无 jti）无吊销能力，靠过期自然失效。
func BlacklistKey(jti string) string { return "jwt:bl:" + jti }

// newJTI 生成随机令牌 ID（crypto/rand 16 字节 hex），不可枚举、不可预测
func newJTI() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
