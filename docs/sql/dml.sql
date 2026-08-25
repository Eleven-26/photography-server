-- =====================================================================
-- SLOT 摄影工作室管理系统 初始化数据脚本 (DML)
-- 依赖 docs/sql/ddl.sql 先执行
-- 默认超级管理员账号：admin / admin123456
-- =====================================================================

USE `photography`;

-- 公司/工作室
INSERT INTO `sys_company` (`created_by`,`updated_by`,`name`,`contact_name`,`contact_phone`,`address`,`status`)
VALUES (1, 1, 'SLOT摄影工作室', '王店长', '13800000000', '上海市静安区某某路88号', 1);

-- 门店
INSERT INTO `sys_store` (`created_by`,`updated_by`,`company_id`,`name`,`address`,`phone`,`status`)
VALUES (1, 1, 1, 'SLOT主门店', '上海市静安区某某路88号', '021-00000000', 1);

-- 角色
INSERT INTO `sys_role` (`created_by`,`updated_by`,`company_id`,`name`,`code`,`remark`,`status`) VALUES
(1, 1, 1, '超级管理员', 'admin', '拥有全部权限', 1),
(1, 1, 1, '店长', 'manager', '门店经营管理', 1),
(1, 1, 1, '摄影师', 'photographer', '拍摄与交付', 1),
(1, 1, 1, '销售', 'sales', '线索与客户跟进', 1);

-- 超级管理员 (密码 admin123456)
INSERT INTO `sys_user` (`created_by`,`updated_by`,`company_id`,`store_id`,`username`,`password`,`nickname`,`mobile`,`role_id`,`status`)
VALUES (1, 1, 1, 1, 'admin', '$2a$10$LInYkTZNMY1PCJT.tFB3Sugee3I5/xj1f1MBS8E6Q2FhwinLYAfES', '超级管理员', '13800000000', 1, 1);

-- 收款方式
INSERT INTO `biz_payment_method` (`created_by`,`updated_by`,`company_id`,`name`,`type`,`account_name`,`account_no`,`status`,`sort`) VALUES
(1, 1, 1, '微信支付', 'wechat', 'SLOT摄影', 'wx_001', 1, 1),
(1, 1, 1, '支付宝', 'alipay', 'SLOT摄影', 'alipay_001', 1, 2),
(1, 1, 1, '银行转账', 'bank', 'SLOT摄影工作室', '6222 0000 0000 0000', 1, 3),
(1, 1, 1, '现金', 'cash', '', '', 1, 4);

-- 套餐
INSERT INTO `biz_package`
(`created_by`,`updated_by`,`company_id`,`store_id`,`code`,`name`,`category`,`base_price`,`deposit_rate`,`deposit_amt`,`photos_included`,`shoot_hours`,`content_desc`,`addon_unit_price`,`status`,`version`,`base_version`,`published_at`)
VALUES
(1, 1, 1, 1, 'PK-001', '婚纱经典套餐', '婚纱', 6999.00, 30.00, 2099.70, 25, 8.00, '含化妆造型2套、外景拍摄、精修25张、赠送全部底片', 100.00, 'active', 1, 0, NOW()),
(1, 1, 1, 1, 'PK-002', '个人写真轻奢套餐', '写真', 2999.00, 30.00, 899.70, 15, 4.00, '含妆造1套、棚拍+外景、精修15张', 80.00, 'active', 1, 0, NOW()),
(1, 1, 1, 1, 'PK-003', '儿童成长套餐', '儿童', 3999.00, 30.00, 1199.70, 20, 5.00, '含主题拍摄、抓拍跟拍、精修20张', 90.00, 'active', 1, 0, NOW());

-- 示例客户
INSERT INTO `crm_customer`
(`created_by`,`updated_by`,`company_id`,`store_id`,`code`,`name`,`mobile`,`wechat`,`gender`,`level`,`source`,`tags`,`status`,`remark`)
VALUES
(1, 1, 1, 1, 'CU-001', '林女士', '13911112222', 'lin2026', 'female', 'gold', '小红书', '婚纱,外景', 'active', '意向12月婚纱拍摄');

-- 示例线索
INSERT INTO `crm_lead`
(`created_by`,`updated_by`,`company_id`,`store_id`,`code`,`customer_id`,`name`,`mobile`,`source`,`project_type`,`budget_min`,`budget_max`,`status`,`shoot_date`,`remark`,`owner_id`)
VALUES
(1, 1, 1, 1, 'LD-001', 1, '林女士', '13911112222', '小红书', '婚纱', 6000.00, 8000.00, 'quoting', '2026-12-12', '看重外景拍摄质量', 1);

-- 示例报价单
INSERT INTO `biz_quote`
(`created_by`,`updated_by`,`company_id`,`code`,`lead_id`,`customer_id`,`package_id`,`version`,`title`,`package_name`,`base_price`,`addon_price`,`total_price`,`status`,`owner_id`,`shoot_date`)
VALUES
(1, 1, 1, 'QT-001', 1, 1, 1, 1, '林女士婚纱报价', '婚纱经典套餐', 6999.00, 0.00, 6999.00, 'sent', 1, '2026-12-12');
