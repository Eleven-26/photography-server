package service

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"time"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"

	"photography-server/internal/domain"
	"photography-server/internal/infrastructure"
	"photography-server/internal/model"
	"photography-server/internal/pkg/errs"
	"photography-server/internal/pkg/jwtpkg"
	"photography-server/internal/pkg/logger"
)

// ClientUser 客户端（H5/小程序）登录上下文，由 CustomerAuth 中间件注入
// 重构 #32：类型别名，实际定义在 domain 包
type ClientUser = domain.ClientUser

// smsCodeKey 验证码 Redis 键前缀，5 分钟过期
const (
	smsCodeKey = "sms:code:%s:%s" // scene:phone
	smsCdKey   = "sms:cd:%s:%s"   // 发送冷却：scene:phone
	smsDayKey  = "sms:day:%s:%s"  // 当日发送次数：scene:phone
	smsFailKey = "sms:fail:%s:%s" // 校验失败次数：scene:phone
	smsCodeTTL = 5 * time.Minute
	smsCdTTL   = 60 * time.Second // 同一手机号发送冷却
	smsDayMax  = 10               // 同一手机号单日发送上限
	smsFailMax = 5                // 校验失败次数上限，超限作废
)

// redis 访问器：统一走 infrastructure 单例（service 不持有基础设施句柄，与分层纪律一致）
func (s *Service) redis() *redis.Client {
	return infrastructure.Redis()
}

// SendSmsCode 发送短信验证码。验证码写入 Redis（5 分钟有效）；
// 实际短信通道未接入（占位），验证码会记录到服务端日志以便联调。
// 限流：同一手机号 60s 冷却 + 单日上限，防止短信轰炸与 Redis 内存被打满。
func (s *Service) SendSmsCode(ctx context.Context, scene, mobile string) error {
	if len(mobile) != 11 {
		return errs.BadRequest("手机号格式错误")
	}
	rdb := s.redis()
	if rdb == nil {
		return errs.Internal("短信服务暂不可用，请稍后再试")
	}

	// 1. 冷却检查：上次发送未满 60s 拒绝
	cdKey := fmt.Sprintf(smsCdKey, scene, mobile)
	if n, err := rdb.Exists(ctx, cdKey).Result(); err == nil && n > 0 {
		return errs.BadRequest("发送过于频繁，请稍后再试")
	}
	// 2. 当日次数检查（含本次，先占位防并发穿透）
	dayKey := fmt.Sprintf(smsDayKey, scene, mobile)
	dayCount, err := rdb.Incr(ctx, dayKey).Result()
	if err != nil {
		return errs.Internal("")
	}
	if dayCount == 1 {
		rdb.Expire(ctx, dayKey, 24*time.Hour)
	}
	if dayCount > smsDayMax {
		return errs.BadRequest("今日发送次数已达上限")
	}

	code, err := genSmsCode()
	if err != nil {
		return errs.Internal("")
	}
	key := fmt.Sprintf(smsCodeKey, scene, mobile)
	if err := rdb.Set(ctx, key, code, smsCodeTTL).Err(); err != nil {
		return errs.Internal("")
	}
	// 3. 写入冷却标记
	rdb.Set(ctx, cdKey, "1", smsCdTTL)
	// TODO(P1): 接入真实短信通道；当前仅记录日志用于开发联调
	fmt.Printf("[sms] scene=%s mobile=%s code=%s\n", scene, mobile, code)
	return nil
}

// verifySmsCode 校验并消费验证码（一次性，原子）。
// 用 GETDEL 原子完成"读取+删除"：并发/重放场景下验证码只会被消费一次，
// 避免旧实现 Del 失败导致同一验证码可重复使用。
func (s *Service) verifySmsCode(ctx context.Context, scene, mobile, code string) error {
	rdb := s.redis()
	if rdb == nil {
		return errs.Internal("短信服务暂不可用，请稍后再试")
	}
	// 失败次数上限：超过则作废验证码，防暴力枚举
	failKey := fmt.Sprintf(smsFailKey, scene, mobile)
	if n, _ := rdb.Get(ctx, failKey).Int(); n >= smsFailMax {
		rdb.Del(ctx, fmt.Sprintf(smsCodeKey, scene, mobile))
		return errs.BadRequest("验证码错误次数过多，请重新获取")
	}

	key := fmt.Sprintf(smsCodeKey, scene, mobile)
	val, err := rdb.GetDel(ctx, key).Result()
	if errors.Is(err, redis.Nil) {
		return errs.BadRequest("验证码错误或已过期")
	}
	if err != nil {
		return errs.Internal("")
	}
	if val != code {
		n, _ := rdb.Incr(ctx, failKey).Result()
		if n == 1 {
			rdb.Expire(ctx, failKey, smsCodeTTL)
		}
		if n >= smsFailMax {
			rdb.Del(ctx, key)
		}
		return errs.BadRequest("验证码错误或已过期")
	}
	// 校验通过：清除失败计数（一次性消费已由 GETDEL 保证）
	rdb.Del(ctx, failKey)
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
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		// 数据库故障等非"不存在"错误直接返回，不能误判为未注册去重复建档
		return nil, "", errs.Internal("")
	}
	if err != nil {
		// 首次登录自动注册客户档案
		c = &model.Customer{
			TenantBase: model.TenantBase{
				Base:      model.Base{CreatedAt: time.Now(), UpdatedAt: time.Now()},
				CompanyID: companyID,
			},
			Code:       domain.GenCode("CU"),
			Name:       maskMobile(mobile),
			Mobile:     mobile,
			Source:     "预约主页",
			Status:     2, // 活跃
			IsVerified: 1,
			OpenID:     openid,
		}
		if err := s.CustomerRepo.Create(ctx, c); err != nil {
			return nil, "", err
		}
	} else {
		updates := map[string]interface{}{"is_verified": 1, "status": 2}
		if openid != "" && c.OpenID == "" {
			updates["openid"] = openid
		}
		if err := s.CustomerRepo.Update(ctx, companyID, c.ID, updates); err != nil {
			logger.Warnf("CustomerSmsLogin: update customer failed, customerID=%d, err=%v", c.ID, err)
		}
		// 登录成功把状态置回活跃：失效认证缓存，避免"流失中"的旧画像在 TTL 内继续拦截
		invalidateCustomerCache(ctx, companyID, c.ID)
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
	if err := s.AuthRepo.TouchLogin(ctx, u.ID, ip); err != nil {
		logger.Warnf("StaffSmsLogin: TouchLogin failed, userID=%d, err=%v", u.ID, err)
	}
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
		if err := s.DeviceRepo.Create(ctx, d); err != nil {
			logger.Warnf("StaffSmsLogin: create device failed, userID=%d, err=%v", u.ID, err)
		}
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
