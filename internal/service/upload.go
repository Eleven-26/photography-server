package service

import (
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
func (s *Service) UploadFile(op Operator, storeID int64, bizType string, bizID int64, fileName string, data []byte) (*UploadResult, error) {
	ext := filepath.Ext(fileName)
	if ext == "" {
		ext = ".bin"
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
	name := genCode("UP") + ext
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
	if err := s.tenant(op).Create(&u).Error; err != nil {
		return nil, err
	}
	return &UploadResult{URL: url, FileName: fileName, FileType: enum.UploadTypeName(fileType), Size: int64(len(data))}, nil
}

func detectFileType(ext string) enum.UploadType {
	switch strings.ToLower(ext) {
	case ".jpg", ".jpeg", ".png", ".gif", ".webp", ".bmp", ".heic":
		return enum.UploadTypeImage
	case ".mp4", ".mov", ".avi", ".mkv", ".webm":
		return enum.UploadTypeVideo
	}
	return enum.UploadTypeFile
}
