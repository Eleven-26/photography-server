-- =====================================================================
-- 增量升级：定制需求接入数据权限 · 2026-09-11
-- 上游基线：upgrade_perm_prune_20260911.sql
--
-- 变更内容：
--   1) biz_custom_request 增加 store_id 列 + 索引（拍板：各门店独立接单）
--   2) 存量回填：已转化记录（lead_id > 0）经 crm_lead.store_id 推断归属
--
-- 归属语义（见 internal/repository/scope.go 的 ScopeCols.Public）：
--   - store_id > 0  —— 门店私有，按数据范围过滤（本店可见/仅本人可见）
--   - store_id = 0  —— 公共池（H5 游客提交、历史无主数据），对任何员工可见，
--                      由员工响应认领；未来 H5 提交页按门店参数化后新数据直接落店。
--   该改造相对旧行为（全公司可见）是收窄，不是放宽。
--
-- 注意：本脚本与 upgrade_perm_prune_20260911.sql 相互独立，无顺序依赖。
-- =====================================================================
USE `photography`;

-- 1. 增加所属门店列
ALTER TABLE `biz_custom_request`
    ADD COLUMN `store_id` bigint NOT NULL DEFAULT '0' COMMENT '所属门店ID(0=公共池未归属)' AFTER `company_id`,
    ADD KEY `idx_custom_req_store` (`store_id`);

-- 2. 存量回填：经转化线索推断门店。
--    已转化（lead_id > 0）的记录按线索所属门店回填；
--    待处理池（lead_id = 0）无归属依据，保持 0 落公共池，由各店响应认领。
UPDATE `biz_custom_request` cr
    JOIN `crm_lead` l ON l.id = cr.lead_id
SET cr.store_id = l.store_id
WHERE cr.lead_id > 0 AND cr.store_id = 0;

-- 校验：回填后门店归属分布（store_id=0 即公共池存量）
SELECT store_id, COUNT(*) AS cnt
FROM biz_custom_request
WHERE deleted = 0
GROUP BY store_id
ORDER BY cnt DESC;
