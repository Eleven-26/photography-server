package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"time"

	"photography-server/internal/enum"
	"photography-server/internal/model"
	"photography-server/internal/pkg/errs"
)

type UploadResult struct {
	URL      string          `json:"url"`
	FileName string          `json:"file_name"`
	FileType enum.UploadType `json:"file_type"` // 1-图片 2-视频 3-文件（与 biz_upload.file_type 同口径）
	Size     int64           `json:"size"`
}

// UploadOptions 上传附加参数。
//
// Public 决定落盘子目录与返回 URL 前缀，二者必须同步切换（路由与磁盘结构一一对应）：
//   - true  → <UploadDir>/media/… 与 /media/…   **免鉴权**，供作品集等对外宣传物料；
//     H5 分享页的浏览者通常未登录，若走鉴权目录图片会整片 401。
//   - false → <UploadDir>/…       与 /uploads/…  强制鉴权，客户隐私数据
//     （订单样片/精修成片/付款凭证/头像）一律留在此处。
//
// 目录物理隔离而非同目录加白名单：静态目录无法按文件做行级鉴权，
// 若把公开文件混在鉴权目录里放开，等于顺带放开该目录下全部隐私文件。
type UploadOptions struct {
	StoreID int64
	BizType string
	BizID   int64
	Public  bool
}

// 公开媒体目录约定（HTTP 侧与磁盘侧同源，勿各写一份字面量）。
//
//	磁盘：<UploadDir>/media/yyyymm/<hex>.<ext>
//	HTTP：/media/yyyymm/<hex>.<ext>   ← router.go 注册静态目录时引用本常量
//
// 两侧若漂移（例如 URL 前缀改了、落盘没跟着改），文件会 404 且症状是"上传成功但图片打不开"，
// 排查成本高，故集中在这里。
const (
	// PublicMediaDir 公开媒体子目录名（相对 UploadDir）
	PublicMediaDir = "media"
	// PublicMediaURLPrefix 公开媒体的 URL 前缀
	PublicMediaURLPrefix = "/media/"
	// PrivateUploadURLPrefix 鉴权媒体的 URL 前缀（订单样片/成片/凭证等隐私文件）
	PrivateUploadURLPrefix = "/uploads/"
)

// uploadTarget 返回落盘子目录（相对 UploadDir）与对应的 URL 前缀。
//
// 独立成纯函数以便单测锁定"公开 / 私有"两条路径的映射 —— 二者必须同步切换，
// 只改一侧会让文件 404；这是本文件最容易被顺手改错的地方，故不内联在 UploadFile 里。
func uploadTarget(public bool, month string) (relDir, urlPrefix string) {
	if public {
		return filepath.Join(PublicMediaDir, month), PublicMediaURLPrefix
	}
	return month, PrivateUploadURLPrefix
}

// UploadFile 保存上传文件到本地 uploads 目录并记录到 biz_upload
// 安全要点：
//  1. 扩展名白名单：只认 filepath.Base 后的尾缀，且必须命中白名单，
//     客户端文件名中的路径穿越（..// 分隔符）经 Base 后自然失效；
//  2. 存储文件名服务端生成（16 字节 crypto/rand → 32 位十六进制），
//     不拼接客户端输入，也无法按日期+短随机枚举扫描他人文件。
func (s *Service) UploadFile(ctx context.Context, op Operator, fileName string, data []byte, opt UploadOptions) (*UploadResult, error) {
	ext := sanitizeExt(fileName)
	if ext == "" {
		return nil, errs.BadRequest(errs.ErrFileTypeUnsupported)
	}
	fileType := detectFileType(ext)
	if fileType == enum.UploadTypeFile {
		// 仅允许图片/视频上传
		return nil, errs.BadRequest(errs.ErrUploadInvalidFile)
	}
	// 按月份分目录防单目录膨胀；公开物料整体下沉到 media/ 子目录（见 UploadOptions 注释）
	month := time.Now().Format("200601")
	rel, prefix := uploadTarget(opt.Public, month)
	dir := filepath.Join(s.UploadDir, rel)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, errs.Internal("")
	}
	name := newUploadName() + ext
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, data, 0644); err != nil {
		return nil, errs.Internal("")
	}
	url := prefix + month + "/" + name

	u := model.Upload{
		TenantBase: model.TenantBase{
			Base:      model.Base{CreatedBy: op.UserID, UpdatedBy: op.UserID},
			CompanyID: op.CompanyID,
		},
		StoreID:  opt.StoreID,
		BizType:  opt.BizType,
		BizID:    opt.BizID,
		FileType: fileType,
		FileName: fileName,
		FileURL:  url,
		FilePath: path,
		Size:     int64(len(data)),
		UploadBy: op.UserID,
	}
	if err := s.UploadRepo.Create(ctx, &u); err != nil {
		return nil, err
	}
	return &UploadResult{URL: url, FileName: fileName, FileType: fileType, Size: int64(len(data))}, nil
}

// sanitizeExt 从客户端文件名提取扩展名：仅取 filepath.Base 的最后一段，
// 命中白名单（纯字母数字、1-10 位、以 . 开头）才返回，否则返回空串拒绝。
// 路径穿越（..、/、\ 等）在 Base 后无法进入 ext，白名单再兜一层。
func sanitizeExt(fileName string) string {
	base := filepath.Base(fileName)
	if base == "." || base == string(filepath.Separator) || base == "" {
		return ""
	}
	ext := strings.ToLower(filepath.Ext(base))
	if len(ext) < 2 || len(ext) > 11 {
		return ""
	}
	for i := 1; i < len(ext); i++ { // 跳过前导 .
		c := ext[i]
		if !(c >= 'a' && c <= 'z' || c >= '0' && c <= '9') {
			return ""
		}
	}
	return ext
}

// newUploadName 生成 32 位十六进制随机文件名（无扩展名），不可枚举、不可预测
func newUploadName() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		// crypto/rand 失败几乎不可能；退化为时间戳，保证函数不失败
		return time.Now().Format("20060102150405.000000000")
	}
	return hex.EncodeToString(b[:])
}

func detectFileType(ext string) enum.UploadType {
	switch ext {
	case ".jpg", ".jpeg", ".png", ".gif", ".webp", ".bmp", ".heic":
		return enum.UploadTypeImage
	case ".mp4", ".mov", ".avi", ".mkv", ".webm":
		return enum.UploadTypeVideo
	}
	return enum.UploadTypeFile
}

// detectFileTypeByURL 按已保存文件的 URL 反推类型，供交付链路使用
// （file_type 不接受客户端上报，避免字符串脏值写进 tinyint 列）。
// URL 形如 /uploads/200601/<32位hex>.<ext>，可能带 query/fragment，先剥离再取扩展名。
func detectFileTypeByURL(rawURL string) enum.UploadType {
	u := rawURL
	if i := strings.IndexAny(u, "?#"); i >= 0 {
		u = u[:i]
	}
	return detectFileType(sanitizeExt(u))
}
