-- =====================================================================
-- SLOT 摄影工作室管理系统 初始化数据脚本 (DML)
-- 依赖 docs/sql/ddl.sql 先执行
-- 默认超级管理员账号：admin / admin123456
-- =====================================================================

USE `photography`;

-- 公司/工作室
INSERT INTO `sys_company` (`name`,`status`,`contact_name`,`contact_phone`,`address`,`created_by`,`updated_by`)
VALUES ('SLOT摄影工作室', 1, '王店长', '13800000000', '上海市静安区某某路88号', 1, 1);

-- 门店
INSERT INTO `sys_store` (`company_id`,`name`,`status`,`address`,`phone`,`created_by`,`updated_by`)
VALUES (1, 'SLOT主门店', 1, '上海市静安区某某路88号', '021-00000000', 1, 1);

-- 角色（data_scope 数据范围：1-全部数据 2-本门店 3-仅本人）
INSERT INTO `sys_role` (`company_id`,`code`,`name`,`status`,`data_scope`,`remark`,`created_by`,`updated_by`) VALUES
(1, 'admin', '超级管理员', 1, 1, '拥有全部权限', 1, 1),
(1, 'manager', '店长', 1, 2, '门店经营管理', 1, 1),
(1, 'photographer', '摄影师', 1, 3, '拍摄与交付', 1, 1),
(1, 'sales', '销售', 1, 2, '线索与客户跟进', 1, 1);

-- 角色权限（RBAC 默认绑定；权限点清单见 internal/domain/perm.go）
-- 各角色权限点数：admin 70 / manager 53 / photographer 28 / sales 32
-- 用 JOIN code 生成 role_id，不写死 ID，保证与增量脚本结果一致
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
    SELECT 'photographer' AS code, 'order:update' AS permission
    UNION ALL
    SELECT 'photographer' AS code, 'order:reschedule' AS permission
    UNION ALL
    SELECT 'photographer' AS code, 'customer:view' AS permission
    UNION ALL
    SELECT 'photographer' AS code, 'lead:view' AS permission
    UNION ALL
    SELECT 'photographer' AS code, 'lead:update' AS permission
    UNION ALL
    SELECT 'photographer' AS code, 'quote:view' AS permission
    UNION ALL
    SELECT 'photographer' AS code, 'package:view' AS permission
    UNION ALL
    SELECT 'photographer' AS code, 'payment:create' AS permission
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
    SELECT 'photographer' AS code, 'settings:view' AS permission
    UNION ALL
    -- settings:update：摄影师在员工端「我的预约主页」维护主页标识（homepage_slug）
    SELECT 'photographer' AS code, 'settings:update' AS permission
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
    SELECT 'sales' AS code, 'request:handle' AS permission
    UNION ALL
    SELECT 'sales' AS code, 'review:view' AS permission
    UNION ALL
    SELECT 'sales' AS code, 'review:reply' AS permission
    UNION ALL
    SELECT 'sales' AS code, 'device:view' AS permission) x ON x.code = r.code
WHERE r.deleted = 0;

-- 超级管理员 (密码 admin123456)
INSERT INTO `sys_user` (`company_id`,`store_id`,`username`,`nickname`,`status`,`role_id`,`password`,`mobile`,`created_by`,`updated_by`)
VALUES (1, 1, 'admin', '超级管理员', 1, 1, '$2a$10$LInYkTZNMY1PCJT.tFB3Sugee3I5/xj1f1MBS8E6Q2FhwinLYAfES', '13800000000', 1, 1);

-- 收款方式
INSERT INTO `biz_payment_method` (`company_id`,`name`,`status`,`type`,`account_name`,`account_no`,`sort`,`created_by`,`updated_by`) VALUES
(1, '微信支付', 1, 'wechat', 'SLOT摄影', 'wx_001', 1, 1, 1),
(1, '支付宝', 1, 'alipay', 'SLOT摄影', 'alipay_001', 2, 1, 1),
(1, '银行转账', 1, 'bank', 'SLOT摄影工作室', '6222 0000 0000 0000', 3, 1, 1),
(1, '现金', 1, 'cash', '', '', 4, 1, 1);

-- 套餐
INSERT INTO `biz_package`
(`company_id`,`store_id`,`code`,`name`,`status`,`category`,`base_price`,`deposit_rate`,`deposit_amt`,`photos_included`,`shoot_hours`,`content_desc`,`addon_unit_price`,`version`,`base_version`,`published_at`,`created_by`,`updated_by`)
VALUES
(1, 1, 'PK-001', '婚纱经典套餐', 1, '婚纱', 6999.00, 30.00, 2099.70, 25, 8.00, '含化妆造型2套、外景拍摄、精修25张、赠送全部底片', 100.00, 1, 0, NOW(), 1, 1),
(1, 1, 'PK-002', '个人写真轻奢套餐', 1, '写真', 2999.00, 30.00, 899.70, 15, 4.00, '含妆造1套、棚拍+外景、精修15张', 80.00, 1, 0, NOW(), 1, 1),
(1, 1, 'PK-003', '儿童成长套餐', 1, '儿童', 3999.00, 30.00, 1199.70, 20, 5.00, '含主题拍摄、抓拍跟拍、精修20张', 90.00, 1, 0, NOW(), 1, 1);

-- 示例客户
INSERT INTO `crm_customer`
(`company_id`,`store_id`,`code`,`name`,`status`,`level`,`source`,`mobile`,`wechat`,`gender`,`tags`,`remark`,`created_by`,`updated_by`)
VALUES
(1, 1, 'CU-001', '林女士', 2, 3, '小红书', '13911112222', 'lin2026', 'female', '婚纱,外景', '意向12月婚纱拍摄', 1, 1);

-- 示例线索
INSERT INTO `crm_lead`
(`company_id`,`store_id`,`code`,`name`,`status`,`source`,`customer_id`,`owner_id`,`project_type`,`mobile`,`budget_min`,`budget_max`,`shoot_date`,`remark`,`created_by`,`updated_by`)
VALUES
(1, 1, 'LD-001', '林女士', 2, '小红书', 1, 1, '婚纱', '13911112222', 6000.00, 8000.00, '2026-12-12', '看重外景拍摄质量', 1, 1);

-- 示例报价单
INSERT INTO `biz_quote`
(`company_id`,`code`,`title`,`status`,`lead_id`,`customer_id`,`package_id`,`owner_id`,`version`,`package_name`,`base_price`,`addon_price`,`total_price`,`shoot_date`,`created_by`,`updated_by`)
VALUES
(1, 'QT-001', '林女士婚纱报价', 2, 1, 1, 1, 1, 1, '婚纱经典套餐', 6999.00, 0.00, 6999.00, '2026-12-12', 1, 1);
