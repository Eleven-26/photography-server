-- =====================================================================
-- 增量升级：权限点清理 · 2026-09-11
-- 上游基线：upgrade_role_permission_20260910.sql
--
-- 变更内容：
--   删除两个「无业务接口挂载」的权限点（2026-09-11 拍板）：
--     - order:price  —— 订单金额无独立修改接口，改价经加项同事务重算，走 order:update
--     - quote:audit  —— 报价无审批接口，状态流转走 quote:update
--   两点此前仅 admin/manager 持有，勾选后无任何实际效果（权限幻觉），一并移除。
--   domain/perm.go 常量与 docs/sql/dml.sql、upgrade_role_permission_20260910.sql
--   的种子已同步删除；本脚本负责清理「已按旧种子初始化」的库。
--
-- 各内置角色权限点数量（清理后）：admin 70 / manager 53 / photographer 27 / sales 32
-- =====================================================================
USE `photography`;

DELETE FROM `sys_role_permission`
WHERE `permission` IN ('order:price', 'quote:audit');

-- 校验：清理后不应再有残留行
SELECT IF(COUNT(*) = 0, 'OK: 残留行已清空', CONCAT('FAIL: 残留 ', COUNT(*), ' 行')) AS prune_check
FROM `sys_role_permission`
WHERE `permission` IN ('order:price', 'quote:audit');
