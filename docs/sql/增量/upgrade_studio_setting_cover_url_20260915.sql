-- =====================================================================
-- 增量升级：工作室设置新增分享封面图 · 2026-09-15
-- 上游基线：增量/ddl-初版.sql
--
-- 变更内容：
--   1) biz_studio_setting 增加 cover_url 列（分享封面图：预约主页 / 分享页顶部大图）
--
-- 语义（与 internal/model/client.go、internal/presentation/dto/client.go 对齐）：
--   - 用途：分享出去后的分享页（H5 C01 首页）顶部大图，即「工作室主页封面」。
--   - 该图属**对外物料**：三端（PC / H5 / 小程序）上传都必须带 public=1，
--     落免鉴权的 /media 目录；否则未登录的分享页浏览者整片 401。
--   - 后端 DTO 该字段为 *string（「非 nil 才更新」语义），因此**支持传空串清空**封面。
--
-- 位置：置于 slogan 之后（与 ddl.sql 的字段顺序一致，字段顺序属一致性校验范围）。
-- 注意：本脚本不保证幂等（重复执行报 Duplicate column，属预期）。执行前请备份。
-- =====================================================================
USE `photography`;

-- 1. 增加分享封面图列
ALTER TABLE `biz_studio_setting`
    ADD COLUMN `cover_url` varchar(500) DEFAULT NULL COMMENT '分享封面图(预约主页/分享页顶部大图)' AFTER `slogan`;

-- 2. 校验：确认列已就位（应返回 1 行，ORDINAL_POSITION 与 ddl.sql 一致）
SELECT ORDINAL_POSITION, COLUMN_NAME, COLUMN_TYPE, IS_NULLABLE, COLUMN_COMMENT
FROM information_schema.COLUMNS
WHERE TABLE_SCHEMA = 'photography'
  AND TABLE_NAME = 'biz_studio_setting'
  AND COLUMN_NAME = 'cover_url';
