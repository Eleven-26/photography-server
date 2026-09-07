package service

import (
	"context"
	"crypto/rand"
	"fmt"
	"math/big"
	"time"

	"github.com/redis/go-redis/v9"

	"photography-server/internal/domain"
	"photography-server/internal/infrastructure"
	"photography-server/internal/model"
	"photography-server/internal/pkg/errs"
	"photography-server/internal/pkg/jwtpkg"
)

// ClientUser 客户端（H5/小程序）登录上下文，由 CustomerAuth 中间件注入
type ClientUser struct {
	CustomerID int64
	CompanyID  int64
	Mobile     string
	Name       string
}

// smsCodeKey 验证码 Redis 键前缀，5 分钟过期
const smsCodeKey = "sms:code:%s:%s" // scene:phone

const smsCodeTTL = 5 * time.Minute

// redis 访问器：统一走 infrastructure 单例（service 不持有基础设施句柄，与分层纪律一致）
func (s *Service) redis() *redis.Client {
	return infrastructure.Redis()
}

// SendSmsCode 发送短信验证码。验证码写入 Redis（5 分钟有效）；
// 实际短信通道未接入（占位），验证码会记录到服务端日志以便联调。
func (s *Service) SendSmsCode(ctx context.Context, scene, mobile string) error {
	if len(mobile) != 11 {
		return errs.BadRequest("手机号格式错误")
	}
	code, err := genSmsCode()
	if err != nil {
		return errs.Internal("")
	}
	key := fmt.Sprintf(smsCodeKey, scene, mobile)
	if err := s.redis().Set(ctx, key, code, smsCodeTTL).Err(); err != nil {
		return errs.Internal("")
	}
	// TODO(P1): 接入真实短信通道；当前仅记录日志用于开发联调
	fmt.Printf("[sms] scene=%s mobile=%s code=%s\n", scene, mobile, code)
	return nil
}

// verifySmsCode 校验并消费验证码（一次性）
func (s *Service) verifySmsCode(ctx context.Context, scene, mobile, code string) error {
	key := fmt.Sprintf(smsCodeKey, scene, mobile)
	val, err := s.redis().Get(ctx, key).Result()
	if err != nil || val != code {
		return errs.BadRequest("验证码错误或已过期")
	}
	s.redis().Del(ctx, key)
	return nil
}

func genSmsCode() (string, error) {
	max := big.NewInt(1000000)
	n, err := rand.Int(rand.Reader, max)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%06d", n.Int64()), nil
}

// CustomerSmsLogin 客户手机号验证码登录（H5/小程序共用）。
// 手机号已存在则直接登录；不存在自动创建客户档案（来源=预约主页）。
func (s *Service) CustomerSmsLogin(ctx context.Context, companyID int64, mobile, code, openid string) (*model.Customer, string, error) {
	if err := s.verifySmsCode(ctx, "login", mobile, code); err != nil {
		return nil, "", err
	}
	c, err := s.CustomerRepo.GetByMobile(ctx, companyID, mobile)
	if err != nil {
		// 首次登录自动注册客户档案
		c = &model.Customer{
			Code:       domain.GenCode("CU"),
			Name:       maskMobile(mobile),
			Mobile:     mobile,
			Source:     "预约主页",
			Status:     2, // 活跃
			IsVerified: 1,
			OpenID:     openid,
		}
		c.CreatedAt = time.Now()
		c.UpdatedAt = time.Now()
		if err := s.CustomerRepo.Create(ctx, c); err != nil {
			return nil, "", err
		}
	} else {
		updates := map[string]interface{}{"is_verified": 1, "status": 2}
		if openid != "" && c.OpenID == "" {
			updates["openid"] = openid
		}
		_ = s.CustomerRepo.Update(ctx, companyID, c.ID, updates)
	}
	token, err := s.customerToken(c)
	if err != nil {
		return nil, "", errs.Internal("")
	}
	return c, token, nil
}

// StaffSmsLogin 摄影师 App 手机号验证码登录（按 sys_user.mobile 匹配员工）
func (s *Service) StaffSmsLogin(ctx context.Context, mobile, code, deviceName, platform, ip string) (*model.SysUser, string, error) {
	if err := s.verifySmsCode(ctx, "login", mobile, code); err != nil {
		return nil, "", err
	}
	u, err := s.AuthRepo.GetByMobile(ctx, mobile)
	if err != nil {
		return nil, "", errs.NotFound("账号不存在，请联系工作室开通")
	}
	if u.Status != 1 {
		return nil, "", errs.Forbidden("账号已被停用")
	}
	token, err := s.staffToken(u)
	if err != nil {
		return nil, "", errs.Internal("")
	}
	_ = s.AuthRepo.TouchLogin(ctx, u.ID, ip)
	if deviceName != "" {
		now := time.Now()
		nowStr := now.Format("2006-01-02 15:04:05")
		d := &model.UserDevice{
			TenantBase: model.TenantBase{
				Base:      model.Base{CreatedAt: now, UpdatedAt: now},
				CompanyID: u.CompanyID,
			},
			UserID:       u.ID,
			DeviceName:   deviceName,
			Platform:     platform,
			LastIP:       ip,
			LastActiveAt: &nowStr,
		}
		_ = s.DeviceRepo.Create(ctx, d)
	}
	return u, token, nil
}

// CustomerTokenClaims 签发客户 JWT（UserType=customer）
func (s *Service) customerToken(c *model.Customer) (string, error) {
	return jwtpkg.Generate(s.JWTSecret, s.JWTIssuer, 24*30, jwtpkg.Claims{
		UserID:    c.ID,
		Username:  maskMobile(c.Mobile),
		CompanyID: c.CompanyID,
		UserType:  jwtpkg.UserTypeCustomer,
	})
}

func (s *Service) staffToken(u *model.SysUser) (string, error) {
	return jwtpkg.Generate(s.JWTSecret, s.JWTIssuer, 24*7, jwtpkg.Claims{
		UserID:    u.ID,
		Username:  u.Username,
		CompanyID: u.CompanyID,
		StoreID:   u.StoreID,
		RoleID:    u.RoleID,
		UserType:  jwtpkg.UserTypeStaff,
	})
}

// ListDevices 登录设备列表
func (s *Service) ListDevices(ctx context.Context, op Operator) ([]model.UserDevice, error) {
	return s.DeviceRepo.ListByUser(ctx, op.CompanyID, op.UserID)
}

// RemoveDevice 踢出登录设备
func (s *Service) RemoveDevice(ctx context.Context, op Operator, id int64) error {
	return s.DeviceRepo.Delete(ctx, op.CompanyID, id)
}

func maskMobile(mobile string) string {
	if len(mobile) != 11 {
		return mobile
	}
	return mobile[:3] + "****" + mobile[7:]
}
