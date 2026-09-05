package repository

import (
	"context"

	"gorm.io/gorm"

	"photography-server/internal/model"
)

type UploadRepo struct {
	Repo
}

// WithTx 返回绑定到指定事务连接的副本，事务内的所有写操作将复用该连接，
// 保证跨多张表的写入原子性（失败自动回滚）。
func (r *UploadRepo) WithTx(tx *gorm.DB) *UploadRepo {
	return &UploadRepo{Repo: Repo{db: tx}}
}

func NewUploadRepo() *UploadRepo { return &UploadRepo{} }

// Create 记录上传文件（company_id 由调用方在 model 上填充）
func (r *UploadRepo) Create(ctx context.Context, u *model.Upload) error {
	return r.conn().WithContext(ctx).Create(u).Error
}
