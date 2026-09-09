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
	URL      string `json:"url"`
	FileName string `json:"file_name"`
	FileType string `json:"file_type"`
	Size     int64  `json:"size"`
}

// UploadFile 保存上传文件到本地 uploads 目录并记录到 biz_upload
// 安全要点：
//  1. 扩展名白名单：只认 filepath.Base 后的尾缀，且必须命中白名单，
//     客户端文件名中的路径穿越（..// 分隔符）经 Base 后自然失效；
//  2. 存储文件名服务端生成（16 字节 crypto/rand → 32 位十六进制），
//     不拼接客户端输入，也无法按日期+短随机枚举扫描他人文件。
func (s *Service) UploadFile(ctx context.Context, op Operator, storeID int64, bizType string, bizID int64, fileName string, data []byte) (*UploadResult, error) {
	ext := sanitizeExt(fileName)
	if ext == "" {
		return nil, errs.BadRequest("不支持的文件类型")
	}
	fileType := detectFileType(ext)
	if fileType == enum.UploadTypeFile {
		// 仅允许图片/视频上传
		return nil, errs.BadRequest("仅支持图片/视频文件")
	}
	dir := filepath.Join(s.UploadDir, time.Now().Format("200601"))
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, errs.Internal("")
	}
	name := newUploadName() + ext
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, data, 0644); err != nil {
		return nil, errs.Internal("")
	}
	url := "/uploads/" + time.Now().Format("200601") + "/" + name

	u := model.Upload{
		TenantBase: model.TenantBase{
			Base:      model.Base{CreatedBy: op.UserID, UpdatedBy: op.UserID},
			CompanyID: op.CompanyID,
		},
		StoreID:  storeID,
		BizType:  bizType,
		BizID:    bizID,
		FileType: enum.UploadTypeName(fileType),
		FileName: fileName,
		FileURL:  url,
		FilePath: path,
		Size:     int64(len(data)),
		UploadBy: op.UserID,
	}
	if err := s.UploadRepo.Create(ctx, &u); err != nil {
		return nil, err
	}
	return &UploadResult{URL: url, FileName: fileName, FileType: enum.UploadTypeName(fileType), Size: int64(len(data))}, nil
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
