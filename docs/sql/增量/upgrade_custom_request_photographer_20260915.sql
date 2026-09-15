-- =====================================================================
-- 增量升级：定制需求新增「指定摄影师」 · 2026-09-15
-- 上游基线：增量/ddl-初版.sql
--
-- 变更内容：
--   1) biz_custom_request 增加 photographer_id 列（客户指定 / 分享链接带入的摄影师）
--   2) biz_custom_request 增加 photographer 列（摄影师姓名快照）
--   3) biz_custom_request 增加 idx_custom_req_photographer 索引（管理端按摄影师筛选）
--
-- 背景（原链路是断的）：
--   分享链接 https://host/?slug=xxx&staff_id=12 的 staff_id 此前**只被「预约下单」消费**
--   （h5.go → BookingSubmit → biz_order.photographer_id）；定制需求提交从未读取它，
--   且本表原本没有摄影师列 —— 客户从谁的链接进来提交需求都无从归属。
--   补齐后：H5 定制需求页把「客户选中的摄影师」或「分享链接带入的摄影师」随单落库，
--   PC / 员工端「定制需求」列表可展示并按摄影师筛选。
--
-- 语义（与 internal/model/client.go、internal/presentation/dto/client.go 对齐）：
--   - photographer_id = 0 表示**未指定**：需求落 store_id 对应门店或公共池，由工作室后续指派/认领；
--     该口径与 biz_order.photographer_id（0=未指派）、本表 store_id（0=公共池）保持一致。
--   - photographer 是**姓名快照**（同 biz_order.photographer 的做法）：列表展示无需回表 join sys_user，
--     员工改名后历史需求仍显示当时的服务摄影师。
--
-- 位置：photographer_id / photographer 置于 store_id 之后（三者同属「归属键」，分组前置）。
-- 索引：photographer_id 是管理端筛选维度，随列一起建索引（命名沿用 idx_custom_req_* 前缀）。
-- ⚠️ 本脚本不保证幂等（重复执行报 Duplicate column / Duplicate key，属预期）。执行前请备份。
-- =====================================================================
USE `photography`;

-- 1. 指定摄影师：归属键 + 姓名快照
ALTER TABLE `biz_custom_request`
    ADD COLUMN `photographer_id` bigint NOT NULL DEFAULT '0' COMMENT '指定摄影师ID(0=未指定，由门店/工作室认领)' AFTER `store_id`,
    ADD COLUMN `photographer` varchar(50) DEFAULT NULL COMMENT '指定摄影师姓名(快照)' AFTER `photographer_id`;

-- 2. 按摄影师筛选的索引（PC / 员工端定制需求列表）
ALTER TABLE `biz_custom_request`
    ADD INDEX `idx_custom_req_photographer` (`photographer_id`);

-- 3. 校验：确认列已就位（应返回 2 行，ORDINAL_POSITION 与 ddl.sql 一致）
SELECT ORDINAL_POSITION, COLUMN_NAME, COLUMN_TYPE, IS_NULLABLE, COLUMN_COMMENT
FROM information_schema.COLUMNS
WHERE TABLE_SCHEMA = 'photography'
  AND TABLE_NAME = 'biz_custom_request'
  AND COLUMN_NAME IN ('photographer_id', 'photographer')
ORDER BY ORDINAL_POSITION;
