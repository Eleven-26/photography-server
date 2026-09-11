-- =====================================================================
-- 增量升级：角色权限管理（RBAC）· 2026-09-10
-- 上游基线：upgrade_p2_customer_pref_20260910.sql
--
-- 变更内容：
--   1) 新增表 sys_role_permission —— 角色与权限点的绑定关系
--   2) sys_role 增加 data_scope 列 —— 行级数据范围 1-全部 2-本门店 3-仅本人
--   3) biz_asset 增加 store_id 列 + 索引，并按创建人所属门店回填存量
--   4) 为 4 个内置角色写入默认权限与默认数据范围（结果须与 dml.sql 一致）
--
-- 说明：
--   - 权限点本身**不入库**（编译期常量，见 internal/domain/perm.go），
--     本表只存「角色 → 权限点」的绑定关系；新增权限点无需改数据。
--   - sys_role_permission 不做软删除：唯一键 uk_role_perm 配合
--     「保存时全量覆盖（物理删旧 + 批量插新）」写入策略，若软删则
--     "删掉某权限再加回来" 会命中旧记录导致唯一键冲突。本表无追溯价值。
--   - 权限种子用 JOIN sys_role ON code 生成 role_id，不写死角色 ID，
--     多租户环境下每个公司的同名内置角色都会获得默认权限。
--   - 本脚本**不保证幂等**：重复执行会报「表/字段已存在」，属预期，请勿重复执行。
-- =====================================================================

USE `photography`;

-- ---------------------------------------------------------------------
-- 1. 新增表：角色权限关联
-- ---------------------------------------------------------------------
DROP TABLE IF EXISTS `sys_role_permission`;
CREATE TABLE `sys_role_permission`
(
    `id`         bigint      NOT NULL AUTO_INCREMENT COMMENT '主键ID',
    `company_id` bigint      NOT NULL DEFAULT '0' COMMENT '公司ID',
    `role_id`    bigint      NOT NULL DEFAULT '0' COMMENT '角色ID',
    `permission` varchar(64) NOT NULL COMMENT '权限点 resource:action，如 order:view',
    `created_at` datetime    NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    PRIMARY KEY (`id`),
    UNIQUE KEY `uk_role_perm` (`company_id`,`role_id`,`permission`),
    KEY          `idx_roleperm_role` (`role_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8 COMMENT='角色权限关联';

-- ---------------------------------------------------------------------
-- 2. sys_role 增加数据范围列
-- ---------------------------------------------------------------------
ALTER TABLE `sys_role`
    ADD COLUMN `data_scope` tinyint NOT NULL DEFAULT '1' COMMENT '数据范围 1-全部数据 2-本门店 3-仅本人' AFTER `status`;

-- ---------------------------------------------------------------------
-- 3. biz_asset 增加所属门店列（补齐全链路门店口径）
--    原因：biz_asset 既无 store_id 也无关联订单，行级过滤只能靠创建人，
--    会让店长看不到店内其他摄影师上传的作品（功能残缺，非权限收紧）。
-- ---------------------------------------------------------------------
ALTER TABLE `biz_asset`
    ADD COLUMN `store_id` bigint NOT NULL DEFAULT '0' COMMENT '所属门店ID' AFTER `company_id`,
    ADD KEY `idx_asset_store` (`store_id`);

-- 存量回填：按创建人所属门店推断。
-- 若创建人已换门店或已停用，回填值可能有偏差，属可接受的历史数据误差；
-- 新数据的 store_id 由业务写入时带上。
UPDATE `biz_asset` a
    JOIN `sys_user` u ON u.id = a.created_by
SET a.store_id = u.store_id
WHERE a.store_id = 0;

-- ---------------------------------------------------------------------
-- 4. 内置角色默认数据范围
--    admin 锁定不可改（代码按角色码短路给全权限）；sales 取「本门店」而非
--    「仅本人」—— 客户交接给新销售后若按仅本人过滤，新人将查不到该客户。
-- ---------------------------------------------------------------------
UPDATE `sys_role` SET `data_scope` = 1 WHERE `code` = 'admin';
UPDATE `sys_role` SET `data_scope` = 2 WHERE `code` = 'manager';
UPDATE `sys_role` SET `data_scope` = 3 WHERE `code` = 'photographer';
UPDATE `sys_role` SET `data_scope` = 2 WHERE `code` = 'sales';

-- ---------------------------------------------------------------------
-- 5. 内置角色默认权限（与 dml.sql 保持一致）
--    先清理内置角色的既有绑定，再整体写入，保证升级结果可预期。
-- ---------------------------------------------------------------------
DELETE rp
FROM `sys_role_permission` rp
         JOIN `sys_role` r ON r.id = rp.role_id
WHERE r.code IN ('admin', 'manager', 'photographer', 'sales');

INSERT INTO `sys_role_permission` (`company_id`, `role_id`, `permission`)
SELECT r.company_id,
       r.id,
       x.permission
FROM `sys_role` r
         JOIN (    SELECT 'admin' AS code, 'dashboard:view' AS permission
    UNION ALL
    SELECT 'admin' AS code, 'order:view' AS permission
    UNION ALL
    SELECT 'admin' AS code, 'order:create' AS permission
    UNION ALL
    SELECT 'admin' AS code, 'order:update' AS permission
    UNION ALL
    SELECT 'admin' AS code, 'order:status' AS permission
    UNION ALL
    SELECT 'admin' AS code, 'order:cancel' AS permission
    UNION ALL
    SELECT 'admin' AS code, 'order:price' AS permission
    UNION ALL
    SELECT 'admin' AS code, 'order:reschedule' AS permission
    UNION ALL
    SELECT 'admin' AS code, 'order:reschedule_audit' AS permission
    UNION ALL
    SELECT 'admin' AS code, 'customer:view' AS permission
    UNION ALL
    SELECT 'admin' AS code, 'customer:create' AS permission
    UNION ALL
    SELECT 'admin' AS code, 'customer:update' AS permission
    UNION ALL
    SELECT 'admin' AS code, 'customer:delete' AS permission
    UNION ALL
    SELECT 'admin' AS code, 'lead:view' AS permission
    UNION ALL
    SELECT 'admin' AS code, 'lead:create' AS permission
    UNION ALL
    SELECT 'admin' AS code, 'lead:update' AS permission
    UNION ALL
    SELECT 'admin' AS code, 'lead:assign' AS permission
    UNION ALL
    SELECT 'admin' AS code, 'lead:convert' AS permission
    UNION ALL
    SELECT 'admin' AS code, 'lead:delete' AS permission
    UNION ALL
    SELECT 'admin' AS code, 'quote:view' AS permission
    UNION ALL
    SELECT 'admin' AS code, 'quote:create' AS permission
    UNION ALL
    SELECT 'admin' AS code, 'quote:update' AS permission
    UNION ALL
    SELECT 'admin' AS code, 'quote:audit' AS permission
    UNION ALL
    SELECT 'admin' AS code, 'package:view' AS permission
    UNION ALL
    SELECT 'admin' AS code, 'package:create' AS permission
    UNION ALL
    SELECT 'admin' AS code, 'package:update' AS permission
    UNION ALL
    SELECT 'admin' AS code, 'package:publish' AS permission
    UNION ALL
    SELECT 'admin' AS code, 'package:delete' AS permission
    UNION ALL
    SELECT 'admin' AS code, 'payment:view' AS permission
    UNION ALL
    SELECT 'admin' AS code, 'payment:create' AS permission
    UNION ALL
    SELECT 'admin' AS code, 'payment:confirm' AS permission
    UNION ALL
    SELECT 'admin' AS code, 'payment:delete' AS permission
    UNION ALL
    SELECT 'admin' AS code, 'refund:view' AS permission
    UNION ALL
    SELECT 'admin' AS code, 'refund:create' AS permission
    UNION ALL
    SELECT 'admin' AS code, 'refund:audit' AS permission
    UNION ALL
    SELECT 'admin' AS code, 'delivery:view' AS permission
    UNION ALL
    SELECT 'admin' AS code, 'delivery:create' AS permission
    UNION ALL
    SELECT 'admin' AS code, 'delivery:update' AS permission
    UNION ALL
    SELECT 'admin' AS code, 'delivery:delete' AS permission
    UNION ALL
    SELECT 'admin' AS code, 'asset:view' AS permission
    UNION ALL
    SELECT 'admin' AS code, 'asset:upload' AS permission
    UNION ALL
    SELECT 'admin' AS code, 'asset:update' AS permission
    UNION ALL
    SELECT 'admin' AS code, 'asset:audit' AS permission
    UNION ALL
    SELECT 'admin' AS code, 'asset:delete' AS permission
    UNION ALL
    SELECT 'admin' AS code, 'calendar:view' AS permission
    UNION ALL
    SELECT 'admin' AS code, 'calendar:update' AS permission
    UNION ALL
    SELECT 'admin' AS code, 'finance:view' AS permission
    UNION ALL
    SELECT 'admin' AS code, 'finance:export' AS permission
    UNION ALL
    SELECT 'admin' AS code, 'settings:view' AS permission
    UNION ALL
    SELECT 'admin' AS code, 'settings:update' AS permission
    UNION ALL
    SELECT 'admin' AS code, 'user:view' AS permission
    UNION ALL
    SELECT 'admin' AS code, 'user:create' AS permission
    UNION ALL
    SELECT 'admin' AS code, 'user:update' AS permission
    UNION ALL
    SELECT 'admin' AS code, 'user:delete' AS permission
    UNION ALL
    SELECT 'admin' AS code, 'user:resetpwd' AS permission
    UNION ALL
    SELECT 'admin' AS code, 'role:view' AS permission
    UNION ALL
    SELECT 'admin' AS code, 'role:create' AS permission
    UNION ALL
    SELECT 'admin' AS code, 'role:update' AS permission
    UNION ALL
    SELECT 'admin' AS code, 'role:delete' AS permission
    UNION ALL
    SELECT 'admin' AS code, 'role:grant' AS permission
    UNION ALL
    SELECT 'admin' AS code, 'store:view' AS permission
    UNION ALL
    SELECT 'admin' AS code, 'store:create' AS permission
    UNION ALL
    SELECT 'admin' AS code, 'store:update' AS permission
    UNION ALL
    SELECT 'admin' AS code, 'store:delete' AS permission
    UNION ALL
    SELECT 'admin' AS code, 'log:view' AS permission
    UNION ALL
    SELECT 'admin' AS code, 'notification:view' AS permission
    UNION ALL
    SELECT 'admin' AS code, 'request:view' AS permission
    UNION ALL
    SELECT 'admin' AS code, 'request:handle' AS permission
    UNION ALL
    SELECT 'admin' AS code, 'review:view' AS permission
    UNION ALL
    SELECT 'admin' AS code, 'review:reply' AS permission
    UNION ALL
    SELECT 'admin' AS code, 'device:view' AS permission
    UNION ALL
    SELECT 'admin' AS code, 'device:manage' AS permission
    UNION ALL
    SELECT 'manager' AS code, 'dashboard:view' AS permission
    UNION ALL
    SELECT 'manager' AS code, 'order:view' AS permission
    UNION ALL
    SELECT 'manager' AS code, 'order:create' AS permission
    UNION ALL
    SELECT 'manager' AS code, 'order:update' AS permission
    UNION ALL
    SELECT 'manager' AS code, 'order:status' AS permission
    UNION ALL
    SELECT 'manager' AS code, 'order:cancel' AS permission
    UNION ALL
    SELECT 'manager' AS code, 'order:price' AS permission
    UNION ALL
    SELECT 'manager' AS code, 'order:reschedule_audit' AS permission
    UNION ALL
    SELECT 'manager' AS code, 'customer:view' AS permission
    UNION ALL
    SELECT 'manager' AS code, 'customer:create' AS permission
    UNION ALL
    SELECT 'manager' AS code, 'customer:update' AS permission
    UNION ALL
    SELECT 'manager' AS code, 'customer:delete' AS permission
    UNION ALL
    SELECT 'manager' AS code, 'lead:view' AS permission
    UNION ALL
    SELECT 'manager' AS code, 'lead:create' AS permission
    UNION ALL
    SELECT 'manager' AS code, 'lead:update' AS permission
    UNION ALL
    SELECT 'manager' AS code, 'lead:assign' AS permission
    UNION ALL
    SELECT 'manager' AS code, 'lead:convert' AS permission
    UNION ALL
    SELECT 'manager' AS code, 'lead:delete' AS permission
    UNION ALL
    SELECT 'manager' AS code, 'quote:view' AS permission
    UNION ALL
    SELECT 'manager' AS code, 'quote:create' AS permission
    UNION ALL
    SELECT 'manager' AS code, 'quote:update' AS permission
    UNION ALL
    SELECT 'manager' AS code, 'quote:audit' AS permission
    UNION ALL
    SELECT 'manager' AS code, 'package:view' AS permission
    UNION ALL
    SELECT 'manager' AS code, 'package:create' AS permission
    UNION ALL
    SELECT 'manager' AS code, 'package:update' AS permission
    UNION ALL
    SELECT 'manager' AS code, 'package:publish' AS permission
    UNION ALL
    SELECT 'manager' AS code, 'payment:view' AS permission
    UNION ALL
    SELECT 'manager' AS code, 'payment:create' AS permission
    UNION ALL
    SELECT 'manager' AS code, 'payment:confirm' AS permission
    UNION ALL
    SELECT 'manager' AS code, 'refund:view' AS permission
    UNION ALL
    SELECT 'manager' AS code, 'refund:create' AS permission
    UNION ALL
    SELECT 'manager' AS code, 'refund:audit' AS permission
    UNION ALL
    SELECT 'manager' AS code, 'delivery:view' AS permission
    UNION ALL
    SELECT 'manager' AS code, 'delivery:create' AS permission
    UNION ALL
    SELECT 'manager' AS code, 'delivery:update' AS permission
    UNION ALL
    SELECT 'manager' AS code, 'delivery:delete' AS permission
    UNION ALL
    SELECT 'manager' AS code, 'asset:view' AS permission
    UNION ALL
    SELECT 'manager' AS code, 'asset:upload' AS permission
    UNION ALL
    SELECT 'manager' AS code, 'asset:update' AS permission
    UNION ALL
    SELECT 'manager' AS code, 'asset:audit' AS permission
    UNION ALL
    SELECT 'manager' AS code, 'asset:delete' AS permission
    UNION ALL
    SELECT 'manager' AS code, 'calendar:view' AS permission
    UNION ALL
    SELECT 'manager' AS code, 'calendar:update' AS permission
    UNION ALL
    SELECT 'manager' AS code, 'finance:view' AS permission
    UNION ALL
    SELECT 'manager' AS code, 'settings:view' AS permission
    UNION ALL
    SELECT 'manager' AS code, 'user:view' AS permission
    UNION ALL
    SELECT 'manager' AS code, 'store:view' AS permission
    UNION ALL
    SELECT 'manager' AS code, 'log:view' AS permission
    UNION ALL
    SELECT 'manager' AS code, 'notification:view' AS permission
    UNION ALL
    SELECT 'manager' AS code, 'request:view' AS permission
    UNION ALL
    SELECT 'manager' AS code, 'request:handle' AS permission
    UNION ALL
    SELECT 'manager' AS code, 'review:view' AS permission
    UNION ALL
    SELECT 'manager' AS code, 'review:reply' AS permission
    UNION ALL
    SELECT 'manager' AS code, 'device:view' AS permission
    UNION ALL
    SELECT 'manager' AS code, 'device:manage' AS permission
    UNION ALL
    SELECT 'photographer' AS code, 'dashboard:view' AS permission
    UNION ALL
    SELECT 'photographer' AS code, 'order:view' AS permission
    UNION ALL
    SELECT 'photographer' AS code, 'order:status' AS permission
    UNION ALL
    SELECT 'photographer' AS code, 'order:reschedule' AS permission
    UNION ALL
    SELECT 'photographer' AS code, 'customer:view' AS permission
    UNION ALL
    SELECT 'photographer' AS code, 'lead:view' AS permission
    UNION ALL
    SELECT 'photographer' AS code, 'quote:view' AS permission
    UNION ALL
    SELECT 'photographer' AS code, 'package:view' AS permission
    UNION ALL
    SELECT 'photographer' AS code, 'delivery:view' AS permission
    UNION ALL
    SELECT 'photographer' AS code, 'delivery:create' AS permission
    UNION ALL
    SELECT 'photographer' AS code, 'delivery:update' AS permission
    UNION ALL
    SELECT 'photographer' AS code, 'delivery:delete' AS permission
    UNION ALL
    SELECT 'photographer' AS code, 'asset:view' AS permission
    UNION ALL
    SELECT 'photographer' AS code, 'asset:upload' AS permission
    UNION ALL
    SELECT 'photographer' AS code, 'asset:update' AS permission
    UNION ALL
    SELECT 'photographer' AS code, 'calendar:view' AS permission
    UNION ALL
    SELECT 'photographer' AS code, 'calendar:update' AS permission
    UNION ALL
    SELECT 'photographer' AS code, 'notification:view' AS permission
    UNION ALL
    SELECT 'photographer' AS code, 'request:view' AS permission
    UNION ALL
    SELECT 'photographer' AS code, 'request:handle' AS permission
    UNION ALL
    SELECT 'photographer' AS code, 'review:view' AS permission
    UNION ALL
    SELECT 'photographer' AS code, 'review:reply' AS permission
    UNION ALL
    SELECT 'photographer' AS code, 'device:view' AS permission
    UNION ALL
    SELECT 'sales' AS code, 'dashboard:view' AS permission
    UNION ALL
    SELECT 'sales' AS code, 'order:view' AS permission
    UNION ALL
    SELECT 'sales' AS code, 'order:create' AS permission
    UNION ALL
    SELECT 'sales' AS code, 'order:update' AS permission
    UNION ALL
    SELECT 'sales' AS code, 'order:status' AS permission
    UNION ALL
    SELECT 'sales' AS code, 'customer:view' AS permission
    UNION ALL
    SELECT 'sales' AS code, 'customer:create' AS permission
    UNION ALL
    SELECT 'sales' AS code, 'customer:update' AS permission
    UNION ALL
    SELECT 'sales' AS code, 'customer:delete' AS permission
    UNION ALL
    SELECT 'sales' AS code, 'lead:view' AS permission
    UNION ALL
    SELECT 'sales' AS code, 'lead:create' AS permission
    UNION ALL
    SELECT 'sales' AS code, 'lead:update' AS permission
    UNION ALL
    SELECT 'sales' AS code, 'lead:assign' AS permission
    UNION ALL
    SELECT 'sales' AS code, 'lead:convert' AS permission
    UNION ALL
    SELECT 'sales' AS code, 'lead:delete' AS permission
    UNION ALL
    SELECT 'sales' AS code, 'quote:view' AS permission
    UNION ALL
    SELECT 'sales' AS code, 'quote:create' AS permission
    UNION ALL
    SELECT 'sales' AS code, 'quote:update' AS permission
    UNION ALL
    SELECT 'sales' AS code, 'package:view' AS permission
    UNION ALL
    SELECT 'sales' AS code, 'payment:view' AS permission
    UNION ALL
    SELECT 'sales' AS code, 'payment:create' AS permission
    UNION ALL
    SELECT 'sales' AS code, 'refund:view' AS permission
    UNION ALL
    SELECT 'sales' AS code, 'refund:create' AS permission
    UNION ALL
    SELECT 'sales' AS code, 'delivery:view' AS permission
    UNION ALL
    SELECT 'sales' AS code, 'asset:view' AS permission
    UNION ALL
    SELECT 'sales' AS code, 'calendar:view' AS permission
    UNION ALL
    SELECT 'sales' AS code, 'notification:view' AS permission
    UNION ALL
    SELECT 'sales' AS code, 'request:view' AS permission
    UNION ALL
    SELECT 'sales' AS code, 'review:view' AS permission
    UNION ALL
    SELECT 'sales' AS code, 'review:reply' AS permission
    UNION ALL
    SELECT 'sales' AS code, 'device:view' AS permission) x ON x.code = r.code
WHERE r.deleted = 0;

-- ---------------------------------------------------------------------
-- 6. 校验（可选，人工确认用）：
--    各内置角色权限点数量应为 admin 72 / manager 55 / photographer 23 / sales 31
-- ---------------------------------------------------------------------
-- SELECT r.code, r.data_scope, COUNT(rp.id) AS perm_count
-- FROM sys_role r LEFT JOIN sys_role_permission rp ON rp.role_id = r.id
-- WHERE r.deleted = 0 GROUP BY r.id, r.code, r.data_scope ORDER BY r.id;
