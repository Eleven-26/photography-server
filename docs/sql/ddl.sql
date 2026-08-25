-- =====================================================================
-- SLOT 摄影工作室管理系统 数据库初始化脚本 (DDL)
-- 库名：photography   字符集：utf8   排序规则：utf8_general_ci
-- 通用约定：
--   1) 每张业务表固定包含：created_by 创建人 / created_at 创建时间 /
--      updated_by 修改人 / updated_at 修改时间 / deleted 是否删除(软删除)
--   2) 每张业务表均含 company_id，用于 SaaS 多租户(一公司多门店、多门店多管理员)
--   3) 所有字段均带中文注释
-- =====================================================================

CREATE DATABASE IF NOT EXISTS `photography` DEFAULT CHARACTER SET utf8 COLLATE utf8_general_ci;
USE `photography`;

SET NAMES utf8mb4;

-- ---------------------------------------------------------------------
-- 一、系统表
-- ---------------------------------------------------------------------

-- 公司/工作室
DROP TABLE IF EXISTS `sys_company`;
CREATE TABLE `sys_company` (
  `id`            BIGINT       NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `created_by`    BIGINT       NOT NULL DEFAULT 0 COMMENT '创建人',
  `created_at`    DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `updated_by`    BIGINT       NOT NULL DEFAULT 0 COMMENT '修改人',
  `updated_at`    DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '修改时间',
  `deleted`       BIGINT       NOT NULL DEFAULT 0 COMMENT '是否删除 0-否 1-是',
  `name`          VARCHAR(100) NOT NULL COMMENT '公司/工作室名称',
  `logo`          VARCHAR(500) DEFAULT NULL COMMENT 'LOGO地址',
  `contact_name`  VARCHAR(50)  DEFAULT NULL COMMENT '联系人',
  `contact_phone` VARCHAR(20)  DEFAULT NULL COMMENT '联系电话',
  `address`       VARCHAR(200) DEFAULT NULL COMMENT '地址',
  `status`        TINYINT      NOT NULL DEFAULT 1 COMMENT '状态 1-正常 0-停用',
  PRIMARY KEY (`id`),
  KEY `idx_company_deleted` (`deleted`)
) ENGINE=InnoDB AUTO_INCREMENT=1 DEFAULT CHARSET=utf8 COLLATE=utf8_general_ci COMMENT='公司/工作室';

-- 门店
DROP TABLE IF EXISTS `sys_store`;
CREATE TABLE `sys_store` (
  `id`            BIGINT       NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `created_by`    BIGINT       NOT NULL DEFAULT 0 COMMENT '创建人',
  `created_at`    DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `updated_by`    BIGINT       NOT NULL DEFAULT 0 COMMENT '修改人',
  `updated_at`    DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '修改时间',
  `deleted`       BIGINT       NOT NULL DEFAULT 0 COMMENT '是否删除 0-否 1-是',
  `company_id`    BIGINT       NOT NULL DEFAULT 0 COMMENT '公司ID',
  `name`          VARCHAR(100) NOT NULL COMMENT '门店名称',
  `address`       VARCHAR(200) DEFAULT NULL COMMENT '门店地址',
  `phone`         VARCHAR(20)  DEFAULT NULL COMMENT '门店电话',
  `status`        TINYINT      NOT NULL DEFAULT 1 COMMENT '状态 1-正常 0-停用',
  PRIMARY KEY (`id`),
  KEY `idx_store_company` (`company_id`),
  KEY `idx_store_deleted` (`deleted`)
) ENGINE=InnoDB AUTO_INCREMENT=1 DEFAULT CHARSET=utf8 COLLATE=utf8_general_ci COMMENT='门店';

-- 角色
DROP TABLE IF EXISTS `sys_role`;
CREATE TABLE `sys_role` (
  `id`            BIGINT       NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `created_by`    BIGINT       NOT NULL DEFAULT 0 COMMENT '创建人',
  `created_at`    DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `updated_by`    BIGINT       NOT NULL DEFAULT 0 COMMENT '修改人',
  `updated_at`    DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '修改时间',
  `deleted`       BIGINT       NOT NULL DEFAULT 0 COMMENT '是否删除 0-否 1-是',
  `company_id`    BIGINT       NOT NULL DEFAULT 0 COMMENT '公司ID',
  `name`          VARCHAR(50)  NOT NULL COMMENT '角色名称',
  `code`          VARCHAR(50)  NOT NULL COMMENT '角色编码 admin-超级管理员 manager-店长 photographer-摄影师 sales-销售',
  `remark`        VARCHAR(200) DEFAULT NULL COMMENT '备注',
  `status`        TINYINT      NOT NULL DEFAULT 1 COMMENT '状态 1-启用 0-停用',
  PRIMARY KEY (`id`),
  KEY `idx_role_company` (`company_id`),
  KEY `idx_role_deleted` (`deleted`)
) ENGINE=InnoDB AUTO_INCREMENT=1 DEFAULT CHARSET=utf8 COLLATE=utf8_general_ci COMMENT='角色';

-- 后台管理员/员工
DROP TABLE IF EXISTS `sys_user`;
CREATE TABLE `sys_user` (
  `id`            BIGINT       NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `created_by`    BIGINT       NOT NULL DEFAULT 0 COMMENT '创建人',
  `created_at`    DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `updated_by`    BIGINT       NOT NULL DEFAULT 0 COMMENT '修改人',
  `updated_at`    DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '修改时间',
  `deleted`       BIGINT       NOT NULL DEFAULT 0 COMMENT '是否删除 0-否 1-是',
  `company_id`    BIGINT       NOT NULL DEFAULT 0 COMMENT '公司ID',
  `store_id`      BIGINT       NOT NULL DEFAULT 0 COMMENT '所属门店ID',
  `username`      VARCHAR(50)  NOT NULL COMMENT '登录账号',
  `password`      VARCHAR(255) NOT NULL COMMENT '登录密码(bcrypt)',
  `nickname`      VARCHAR(50)  DEFAULT NULL COMMENT '姓名/昵称',
  `mobile`        VARCHAR(20)  DEFAULT NULL COMMENT '手机号',
  `email`         VARCHAR(100) DEFAULT NULL COMMENT '邮箱',
  `avatar`        VARCHAR(500) DEFAULT NULL COMMENT '头像地址',
  `role_id`       BIGINT       NOT NULL DEFAULT 0 COMMENT '角色ID',
  `status`        TINYINT      NOT NULL DEFAULT 1 COMMENT '状态 1-启用 0-停用',
  `last_login_at` DATETIME     DEFAULT NULL COMMENT '最近登录时间',
  `last_login_ip` VARCHAR(50)  DEFAULT NULL COMMENT '最近登录IP',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_user_username` (`username`, `deleted`),
  KEY `idx_user_company` (`company_id`),
  KEY `idx_user_store` (`store_id`),
  KEY `idx_user_role` (`role_id`),
  KEY `idx_user_deleted` (`deleted`)
) ENGINE=InnoDB AUTO_INCREMENT=1 DEFAULT CHARSET=utf8 COLLATE=utf8_general_ci COMMENT='后台管理员/员工';

-- 操作日志
DROP TABLE IF EXISTS `sys_operation_log`;
CREATE TABLE `sys_operation_log` (
  `id`            BIGINT       NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `created_by`    BIGINT       NOT NULL DEFAULT 0 COMMENT '创建人',
  `created_at`    DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `updated_by`    BIGINT       NOT NULL DEFAULT 0 COMMENT '修改人',
  `updated_at`    DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '修改时间',
  `deleted`       BIGINT       NOT NULL DEFAULT 0 COMMENT '是否删除 0-否 1-是',
  `company_id`    BIGINT       NOT NULL DEFAULT 0 COMMENT '公司ID',
  `user_id`       BIGINT       NOT NULL DEFAULT 0 COMMENT '操作人ID',
  `username`      VARCHAR(50)  DEFAULT NULL COMMENT '操作人账号',
  `module`        VARCHAR(50)  DEFAULT NULL COMMENT '模块',
  `action`        VARCHAR(100) DEFAULT NULL COMMENT '操作行为',
  `method`        VARCHAR(10)  DEFAULT NULL COMMENT '请求方法',
  `path`          VARCHAR(200) DEFAULT NULL COMMENT '请求路径',
  `params`        TEXT         COMMENT '请求参数',
  `ip`            VARCHAR(50)  DEFAULT NULL COMMENT '请求IP',
  `status`        TINYINT      DEFAULT NULL COMMENT '状态 1-成功 0-失败',
  `duration`      BIGINT       DEFAULT NULL COMMENT '耗时(毫秒)',
  PRIMARY KEY (`id`),
  KEY `idx_oplog_user` (`user_id`),
  KEY `idx_oplog_company` (`company_id`),
  KEY `idx_oplog_created` (`created_at`),
  KEY `idx_oplog_deleted` (`deleted`)
) ENGINE=InnoDB AUTO_INCREMENT=1 DEFAULT CHARSET=utf8 COLLATE=utf8_general_ci COMMENT='操作日志';

-- 站内通知
DROP TABLE IF EXISTS `sys_notification`;
CREATE TABLE `sys_notification` (
  `id`            BIGINT       NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `created_by`    BIGINT       NOT NULL DEFAULT 0 COMMENT '创建人',
  `created_at`    DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `updated_by`    BIGINT       NOT NULL DEFAULT 0 COMMENT '修改人',
  `updated_at`    DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '修改时间',
  `deleted`       BIGINT       NOT NULL DEFAULT 0 COMMENT '是否删除 0-否 1-是',
  `company_id`    BIGINT       NOT NULL DEFAULT 0 COMMENT '公司ID',
  `receiver_id`   BIGINT       NOT NULL DEFAULT 0 COMMENT '接收人ID',
  `type`          VARCHAR(20)  DEFAULT NULL COMMENT '类型 order-订单 finance-财务 system-系统',
  `title`         VARCHAR(100) DEFAULT NULL COMMENT '标题',
  `content`       VARCHAR(500) DEFAULT NULL COMMENT '内容',
  `biz_type`      VARCHAR(20)  DEFAULT NULL COMMENT '业务类型 order-订单 refund-退款',
  `biz_id`        BIGINT       DEFAULT NULL COMMENT '业务ID',
  `is_read`       TINYINT      NOT NULL DEFAULT 0 COMMENT '是否已读 0-未读 1-已读',
  `read_at`       DATETIME     DEFAULT NULL COMMENT '已读时间',
  PRIMARY KEY (`id`),
  KEY `idx_notice_receiver` (`receiver_id`),
  KEY `idx_notice_company` (`company_id`),
  KEY `idx_notice_read` (`is_read`),
  KEY `idx_notice_deleted` (`deleted`)
) ENGINE=InnoDB AUTO_INCREMENT=1 DEFAULT CHARSET=utf8 COLLATE=utf8_general_ci COMMENT='站内通知';

-- ---------------------------------------------------------------------
-- 二、客户与线索
-- ---------------------------------------------------------------------

-- 客户
DROP TABLE IF EXISTS `crm_customer`;
CREATE TABLE `crm_customer` (
  `id`            BIGINT         NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `created_by`    BIGINT         NOT NULL DEFAULT 0 COMMENT '创建人',
  `created_at`    DATETIME       NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `updated_by`    BIGINT         NOT NULL DEFAULT 0 COMMENT '修改人',
  `updated_at`    DATETIME       NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '修改时间',
  `deleted`       BIGINT         NOT NULL DEFAULT 0 COMMENT '是否删除 0-否 1-是',
  `company_id`    BIGINT         NOT NULL DEFAULT 0 COMMENT '公司ID',
  `code`          VARCHAR(20)    NOT NULL COMMENT '客户编号 CU-xxx',
  `store_id`      BIGINT         NOT NULL DEFAULT 0 COMMENT '所属门店ID',
  `name`          VARCHAR(50)    NOT NULL COMMENT '客户姓名',
  `mobile`        VARCHAR(20)    DEFAULT NULL COMMENT '手机号',
  `wechat`        VARCHAR(50)    DEFAULT NULL COMMENT '微信号',
  `gender`        VARCHAR(10)    DEFAULT NULL COMMENT '性别 male-男 female-女 unknown-未知',
  `birthday`      VARCHAR(20)    DEFAULT NULL COMMENT '生日',
  `level`         VARCHAR(20)    NOT NULL DEFAULT 'normal' COMMENT '客户等级 normal-普通 gold-黄金 platinum-铂金 diamond-钻石',
  `source`        VARCHAR(50)    DEFAULT NULL COMMENT '客户来源',
  `tags`          VARCHAR(200)   DEFAULT NULL COMMENT '标签(逗号分隔)',
  `status`        VARCHAR(20)    NOT NULL DEFAULT 'potential' COMMENT '状态 potential-潜在 active-活跃 inactive-流失',
  `remark`        VARCHAR(500)   DEFAULT NULL COMMENT '备注',
  `avatar`        VARCHAR(500)   DEFAULT NULL COMMENT '头像地址',
  `order_count`   BIGINT         NOT NULL DEFAULT 0 COMMENT '订单数(冗余)',
  `total_amount`  DECIMAL(12,2)  NOT NULL DEFAULT 0.00 COMMENT '累计消费(冗余)',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_customer_code` (`code`, `deleted`),
  KEY `idx_customer_mobile` (`mobile`),
  KEY `idx_customer_company` (`company_id`),
  KEY `idx_customer_store` (`store_id`),
  KEY `idx_customer_deleted` (`deleted`)
) ENGINE=InnoDB AUTO_INCREMENT=1 DEFAULT CHARSET=utf8 COLLATE=utf8_general_ci COMMENT='客户';

-- 线索
DROP TABLE IF EXISTS `crm_lead`;
CREATE TABLE `crm_lead` (
  `id`             BIGINT        NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `created_by`     BIGINT        NOT NULL DEFAULT 0 COMMENT '创建人',
  `created_at`     DATETIME      NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `updated_by`     BIGINT        NOT NULL DEFAULT 0 COMMENT '修改人',
  `updated_at`     DATETIME      NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '修改时间',
  `deleted`        BIGINT        NOT NULL DEFAULT 0 COMMENT '是否删除 0-否 1-是',
  `company_id`     BIGINT        NOT NULL DEFAULT 0 COMMENT '公司ID',
  `code`           VARCHAR(20)   NOT NULL COMMENT '线索编号 LD-xxx',
  `store_id`       BIGINT        NOT NULL DEFAULT 0 COMMENT '所属门店ID',
  `customer_id`    BIGINT        NOT NULL DEFAULT 0 COMMENT '关联客户ID',
  `name`           VARCHAR(50)   DEFAULT NULL COMMENT '客户姓名',
  `mobile`         VARCHAR(20)   DEFAULT NULL COMMENT '手机号',
  `source`         VARCHAR(50)   DEFAULT NULL COMMENT '线索来源',
  `project_type`   VARCHAR(50)   DEFAULT NULL COMMENT '意向项目类型(婚纱/写真/儿童/全家福/活动跟拍等)',
  `budget_min`     DECIMAL(12,2) DEFAULT NULL COMMENT '预算区间-下限',
  `budget_max`     DECIMAL(12,2) DEFAULT NULL COMMENT '预算区间-上限',
  `status`         VARCHAR(20)   NOT NULL DEFAULT 'pending' COMMENT '状态 pending-待回复 quoting-待报价 quoted-已报价 confirmed-已成交 lose-已流失',
  `shoot_date`     VARCHAR(20)   DEFAULT NULL COMMENT '意向拍摄日期',
  `remark`         VARCHAR(500)  DEFAULT NULL COMMENT '备注',
  `owner_id`       BIGINT        NOT NULL DEFAULT 0 COMMENT '负责人ID',
  `next_follow_at` DATETIME      DEFAULT NULL COMMENT '下次跟进时间',
  `follower`       INT           NOT NULL DEFAULT 0 COMMENT '跟进次数',
  `last_follow_at` DATETIME      DEFAULT NULL COMMENT '最近跟进时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_lead_code` (`code`, `deleted`),
  KEY `idx_lead_company` (`company_id`),
  KEY `idx_lead_store` (`store_id`),
  KEY `idx_lead_customer` (`customer_id`),
  KEY `idx_lead_owner` (`owner_id`),
  KEY `idx_lead_status` (`status`),
  KEY `idx_lead_deleted` (`deleted`)
) ENGINE=InnoDB AUTO_INCREMENT=1 DEFAULT CHARSET=utf8 COLLATE=utf8_general_ci COMMENT='线索';

-- 报价单
DROP TABLE IF EXISTS `biz_quote`;
CREATE TABLE `biz_quote` (
  `id`            BIGINT         NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `created_by`    BIGINT         NOT NULL DEFAULT 0 COMMENT '创建人',
  `created_at`    DATETIME       NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `updated_by`    BIGINT         NOT NULL DEFAULT 0 COMMENT '修改人',
  `updated_at`    DATETIME       NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '修改时间',
  `deleted`       BIGINT         NOT NULL DEFAULT 0 COMMENT '是否删除 0-否 1-是',
  `company_id`    BIGINT         NOT NULL DEFAULT 0 COMMENT '公司ID',
  `code`          VARCHAR(20)    NOT NULL COMMENT '报价单编号 QT-xxx',
  `lead_id`       BIGINT         NOT NULL DEFAULT 0 COMMENT '关联线索ID',
  `customer_id`   BIGINT         NOT NULL DEFAULT 0 COMMENT '关联客户ID',
  `package_id`    BIGINT         NOT NULL DEFAULT 0 COMMENT '关联套餐ID',
  `version`       INT            NOT NULL DEFAULT 1 COMMENT '套餐版本号',
  `title`         VARCHAR(100)   DEFAULT NULL COMMENT '报价标题',
  `package_name`  VARCHAR(100)   DEFAULT NULL COMMENT '套餐名称(下单时快照)',
  `base_price`    DECIMAL(12,2)  DEFAULT NULL COMMENT '基础套餐价(快照)',
  `addon_price`   DECIMAL(12,2)  NOT NULL DEFAULT 0.00 COMMENT '加选金额(快照)',
  `total_price`   DECIMAL(12,2)  DEFAULT NULL COMMENT '报价总额',
  `status`        VARCHAR(20)    NOT NULL DEFAULT 'draft' COMMENT '状态 draft-草稿 sent-已发送 accepted-已接受 rejected-已拒绝 converted-已成交',
  `remark`        VARCHAR(500)   DEFAULT NULL COMMENT '备注',
  `owner_id`      BIGINT         NOT NULL DEFAULT 0 COMMENT '负责人ID',
  `shoot_date`    VARCHAR(20)    DEFAULT NULL COMMENT '意向拍摄日期',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_quote_code` (`code`, `deleted`),
  KEY `idx_quote_company` (`company_id`),
  KEY `idx_quote_lead` (`lead_id`),
  KEY `idx_quote_customer` (`customer_id`),
  KEY `idx_quote_package` (`package_id`),
  KEY `idx_quote_deleted` (`deleted`)
) ENGINE=InnoDB AUTO_INCREMENT=1 DEFAULT CHARSET=utf8 COLLATE=utf8_general_ci COMMENT='报价单';

-- ---------------------------------------------------------------------
-- 三、套餐
-- ---------------------------------------------------------------------

-- 套餐（版本化管理：被订单引用后改价需生成新版本）
DROP TABLE IF EXISTS `biz_package`;
CREATE TABLE `biz_package` (
  `id`               BIGINT         NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `created_by`       BIGINT         NOT NULL DEFAULT 0 COMMENT '创建人',
  `created_at`       DATETIME       NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `updated_by`       BIGINT         NOT NULL DEFAULT 0 COMMENT '修改人',
  `updated_at`       DATETIME       NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '修改时间',
  `deleted`          BIGINT         NOT NULL DEFAULT 0 COMMENT '是否删除 0-否 1-是',
  `company_id`       BIGINT         NOT NULL DEFAULT 0 COMMENT '公司ID',
  `code`             VARCHAR(20)    NOT NULL COMMENT '套餐编号 PK-xxx',
  `store_id`         BIGINT         NOT NULL DEFAULT 0 COMMENT '所属门店ID',
  `name`             VARCHAR(100)   NOT NULL COMMENT '套餐名称',
  `cover`            VARCHAR(500)   DEFAULT NULL COMMENT '套餐封面图',
  `category`         VARCHAR(50)    DEFAULT NULL COMMENT '套餐类型(婚纱/写真/儿童/全家福/活动跟拍等)',
  `base_price`       DECIMAL(12,2)  DEFAULT NULL COMMENT '基础套餐价',
  `deposit_rate`     DECIMAL(5,2)   NOT NULL DEFAULT 30.00 COMMENT '定金比例(%)',
  `deposit_amt`      DECIMAL(12,2)  DEFAULT NULL COMMENT '定金金额(基础价×比例)',
  `photos_included`  INT            NOT NULL DEFAULT 0 COMMENT '包含精修张数',
  `shoot_hours`      DECIMAL(5,2)   NOT NULL DEFAULT 0 COMMENT '拍摄时长(小时)',
  `content_desc`     TEXT           COMMENT '包含内容说明',
  `addon_unit_price` DECIMAL(12,2)  NOT NULL DEFAULT 0.00 COMMENT '加选精修单价',
  `status`           VARCHAR(20)    NOT NULL DEFAULT 'draft' COMMENT '状态 active-已上架 draft-草稿 offline-已下线',
  `version`          INT            NOT NULL DEFAULT 1 COMMENT '套餐版本号',
  `base_version`     INT            NOT NULL DEFAULT 0 COMMENT '上一版本号(0表示初始版本)',
  `published_at`     DATETIME       DEFAULT NULL COMMENT '上架时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_package_code` (`code`, `deleted`),
  KEY `idx_package_company` (`company_id`),
  KEY `idx_package_store` (`store_id`),
  KEY `idx_package_status` (`status`),
  KEY `idx_package_deleted` (`deleted`)
) ENGINE=InnoDB AUTO_INCREMENT=1 DEFAULT CHARSET=utf8 COLLATE=utf8_general_ci COMMENT='套餐';

-- ---------------------------------------------------------------------
-- 四、订单 / 收款 / 退款 / 订单日志
-- ---------------------------------------------------------------------

-- 订单
DROP TABLE IF EXISTS `biz_order`;
CREATE TABLE `biz_order` (
  `id`             BIGINT         NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `created_by`     BIGINT         NOT NULL DEFAULT 0 COMMENT '创建人',
  `created_at`     DATETIME       NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `updated_by`     BIGINT         NOT NULL DEFAULT 0 COMMENT '修改人',
  `updated_at`     DATETIME       NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '修改时间',
  `deleted`        BIGINT         NOT NULL DEFAULT 0 COMMENT '是否删除 0-否 1-是',
  `company_id`     BIGINT         NOT NULL DEFAULT 0 COMMENT '公司ID',
  `code`           VARCHAR(30)    NOT NULL COMMENT '订单编号 SL-xxx',
  `store_id`       BIGINT         NOT NULL DEFAULT 0 COMMENT '所属门店ID',
  `customer_id`    BIGINT         NOT NULL DEFAULT 0 COMMENT '客户ID',
  `customer_name`  VARCHAR(50)    DEFAULT NULL COMMENT '客户姓名(快照)',
  `customer_mobile` VARCHAR(20)   DEFAULT NULL COMMENT '客户手机号(快照)',
  `lead_id`        BIGINT         NOT NULL DEFAULT 0 COMMENT '来源线索ID',
  `quote_id`       BIGINT         NOT NULL DEFAULT 0 COMMENT '来源报价单ID',
  `package_id`     BIGINT         NOT NULL DEFAULT 0 COMMENT '套餐ID',
  `package_name`   VARCHAR(100)   DEFAULT NULL COMMENT '套餐名称(快照)',
  `package_version` INT           NOT NULL DEFAULT 1 COMMENT '下单套餐版本号',
  `base_price`     DECIMAL(12,2)  DEFAULT NULL COMMENT '基础套餐价(快照)',
  `addon_amount`   DECIMAL(12,2)  NOT NULL DEFAULT 0.00 COMMENT '加选精修金额(全部进尾款)',
  `deposit_amt`    DECIMAL(12,2)  NOT NULL DEFAULT 0.00 COMMENT '定金金额(基础价×定金比例)',
  `final_amt`      DECIMAL(12,2)  NOT NULL DEFAULT 0.00 COMMENT '尾款金额(基础价-定金+加选)',
  `total_amt`      DECIMAL(12,2)  NOT NULL DEFAULT 0.00 COMMENT '订单总额',
  `paid_amt`       DECIMAL(12,2)  NOT NULL DEFAULT 0.00 COMMENT '已收款金额',
  `refund_amt`     DECIMAL(12,2)  NOT NULL DEFAULT 0.00 COMMENT '已退款金额',
  `status`         VARCHAR(20)    NOT NULL DEFAULT 'pending_deposit' COMMENT '订单状态 pending_deposit-待定金 scheduled-待拍摄 shooting-拍摄中 retouching-精修中 awaiting_delivery-待交付 completed-已完成 cancelled-已取消',
  `payment_status` VARCHAR(20)    NOT NULL DEFAULT 'pending' COMMENT '支付状态 pending-待核验 confirmed-已确认 unpaid-待支付 refunded-已退款',
  `shoot_date`     VARCHAR(20)    DEFAULT NULL COMMENT '拍摄日期',
  `shoot_time`     VARCHAR(20)    DEFAULT NULL COMMENT '拍摄时间段',
  `shoot_address`  VARCHAR(200)   DEFAULT NULL COMMENT '拍摄地点',
  `photographer_id` BIGINT        NOT NULL DEFAULT 0 COMMENT '摄影师ID',
  `photographer`   VARCHAR(50)    DEFAULT NULL COMMENT '摄影师姓名',
  `remark`         VARCHAR(500)   DEFAULT NULL COMMENT '备注',
  `cancel_reason`  VARCHAR(200)   DEFAULT NULL COMMENT '取消原因',
  `finished_at`    DATETIME       DEFAULT NULL COMMENT '完成时间',
  `owner_id`       BIGINT         NOT NULL DEFAULT 0 COMMENT '负责人(销售)ID',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_order_code` (`code`, `deleted`),
  KEY `idx_order_company` (`company_id`),
  KEY `idx_order_store` (`store_id`),
  KEY `idx_order_customer` (`customer_id`),
  KEY `idx_order_status` (`status`),
  KEY `idx_order_payment_status` (`payment_status`),
  KEY `idx_order_shoot_date` (`shoot_date`),
  KEY `idx_order_owner` (`owner_id`),
  KEY `idx_order_deleted` (`deleted`)
) ENGINE=InnoDB AUTO_INCREMENT=1 DEFAULT CHARSET=utf8 COLLATE=utf8_general_ci COMMENT='订单';

-- 收款记录
DROP TABLE IF EXISTS `biz_order_payment`;
CREATE TABLE `biz_order_payment` (
  `id`            BIGINT         NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `created_by`    BIGINT         NOT NULL DEFAULT 0 COMMENT '创建人',
  `created_at`    DATETIME       NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `updated_by`    BIGINT         NOT NULL DEFAULT 0 COMMENT '修改人',
  `updated_at`    DATETIME       NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '修改时间',
  `deleted`       BIGINT         NOT NULL DEFAULT 0 COMMENT '是否删除 0-否 1-是',
  `company_id`    BIGINT         NOT NULL DEFAULT 0 COMMENT '公司ID',
  `order_id`      BIGINT         NOT NULL DEFAULT 0 COMMENT '订单ID',
  `code`          VARCHAR(20)    NOT NULL COMMENT '收款单号 PM-xxx',
  `customer_id`   BIGINT         NOT NULL DEFAULT 0 COMMENT '客户ID',
  `type`          VARCHAR(20)    DEFAULT NULL COMMENT '收款类型 deposit-定金 final-尾款 addon-加选',
  `amount`        DECIMAL(12,2)  NOT NULL COMMENT '收款金额',
  `method_id`     BIGINT         NOT NULL DEFAULT 0 COMMENT '收款方式ID',
  `method_name`   VARCHAR(50)    DEFAULT NULL COMMENT '收款方式名称(快照)',
  `status`        VARCHAR(20)    NOT NULL DEFAULT 'pending' COMMENT '状态 pending-待核验 confirmed-已确认 refunded-已退款',
  `paid_at`       DATETIME       DEFAULT NULL COMMENT '收款时间',
  `voucher`       VARCHAR(500)   DEFAULT NULL COMMENT '收款凭证图片',
  `operator_id`   BIGINT         NOT NULL DEFAULT 0 COMMENT '收款操作人ID',
  `operator_name` VARCHAR(50)    DEFAULT NULL COMMENT '收款操作人',
  `remark`        VARCHAR(200)   DEFAULT NULL COMMENT '备注',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_payment_code` (`code`, `deleted`),
  KEY `idx_payment_order` (`order_id`),
  KEY `idx_payment_company` (`company_id`),
  KEY `idx_payment_customer` (`customer_id`),
  KEY `idx_payment_status` (`status`),
  KEY `idx_payment_deleted` (`deleted`)
) ENGINE=InnoDB AUTO_INCREMENT=1 DEFAULT CHARSET=utf8 COLLATE=utf8_general_ci COMMENT='收款记录';

-- 退款记录
DROP TABLE IF EXISTS `biz_order_refund`;
CREATE TABLE `biz_order_refund` (
  `id`            BIGINT         NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `created_by`    BIGINT         NOT NULL DEFAULT 0 COMMENT '创建人',
  `created_at`    DATETIME       NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `updated_by`    BIGINT         NOT NULL DEFAULT 0 COMMENT '修改人',
  `updated_at`    DATETIME       NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '修改时间',
  `deleted`       BIGINT         NOT NULL DEFAULT 0 COMMENT '是否删除 0-否 1-是',
  `company_id`    BIGINT         NOT NULL DEFAULT 0 COMMENT '公司ID',
  `order_id`      BIGINT         NOT NULL DEFAULT 0 COMMENT '订单ID',
  `code`          VARCHAR(20)    NOT NULL COMMENT '退款单号 RF-xxx',
  `customer_id`   BIGINT         NOT NULL DEFAULT 0 COMMENT '客户ID',
  `amount`        DECIMAL(12,2)  NOT NULL COMMENT '退款金额',
  `reason`        VARCHAR(200)   DEFAULT NULL COMMENT '退款原因',
  `refund_rule`   VARCHAR(50)    DEFAULT NULL COMMENT '退款规则档位(按拍摄前小时)',
  `status`        VARCHAR(20)    NOT NULL DEFAULT 'applying' COMMENT '状态 applying-申请中 approved-已通过 done-已退款 rejected-已驳回',
  `apply_by`      BIGINT         NOT NULL DEFAULT 0 COMMENT '申请人ID',
  `apply_name`    VARCHAR(50)    DEFAULT NULL COMMENT '申请人',
  `audit_by`      BIGINT         NOT NULL DEFAULT 0 COMMENT '审核人ID',
  `audit_at`      DATETIME       DEFAULT NULL COMMENT '审核时间',
  `audit_remark`  VARCHAR(200)   DEFAULT NULL COMMENT '审核备注',
  `refund_at`     DATETIME       DEFAULT NULL COMMENT '退款时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_refund_code` (`code`, `deleted`),
  KEY `idx_refund_order` (`order_id`),
  KEY `idx_refund_company` (`company_id`),
  KEY `idx_refund_customer` (`customer_id`),
  KEY `idx_refund_status` (`status`),
  KEY `idx_refund_deleted` (`deleted`)
) ENGINE=InnoDB AUTO_INCREMENT=1 DEFAULT CHARSET=utf8 COLLATE=utf8_general_ci COMMENT='退款记录';

-- 订单操作日志
DROP TABLE IF EXISTS `biz_order_log`;
CREATE TABLE `biz_order_log` (
  `id`            BIGINT        NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `created_by`    BIGINT        NOT NULL DEFAULT 0 COMMENT '创建人',
  `created_at`    DATETIME      NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `updated_by`    BIGINT        NOT NULL DEFAULT 0 COMMENT '修改人',
  `updated_at`    DATETIME      NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '修改时间',
  `deleted`       BIGINT        NOT NULL DEFAULT 0 COMMENT '是否删除 0-否 1-是',
  `company_id`    BIGINT        NOT NULL DEFAULT 0 COMMENT '公司ID',
  `order_id`      BIGINT        NOT NULL DEFAULT 0 COMMENT '订单ID',
  `action`        VARCHAR(50)   DEFAULT NULL COMMENT '操作动作',
  `from_status`   VARCHAR(20)   DEFAULT NULL COMMENT '原状态',
  `to_status`     VARCHAR(20)   DEFAULT NULL COMMENT '新状态',
  `content`       VARCHAR(500)  DEFAULT NULL COMMENT '操作内容',
  `operator_id`   BIGINT        NOT NULL DEFAULT 0 COMMENT '操作人ID',
  `operator_name` VARCHAR(50)   DEFAULT NULL COMMENT '操作人',
  PRIMARY KEY (`id`),
  KEY `idx_orderlog_order` (`order_id`),
  KEY `idx_orderlog_company` (`company_id`),
  KEY `idx_orderlog_deleted` (`deleted`)
) ENGINE=InnoDB AUTO_INCREMENT=1 DEFAULT CHARSET=utf8 COLLATE=utf8_general_ci COMMENT='订单操作日志';

-- ---------------------------------------------------------------------
-- 五、交付 / 作品 / 档期 / 收款方式 / 上传
-- ---------------------------------------------------------------------

-- 交付单（选片精修交付流程）
DROP TABLE IF EXISTS `biz_delivery`;
CREATE TABLE `biz_delivery` (
  `id`              BIGINT       NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `created_by`      BIGINT       NOT NULL DEFAULT 0 COMMENT '创建人',
  `created_at`      DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `updated_by`      BIGINT       NOT NULL DEFAULT 0 COMMENT '修改人',
  `updated_at`      DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '修改时间',
  `deleted`         BIGINT       NOT NULL DEFAULT 0 COMMENT '是否删除 0-否 1-是',
  `company_id`      BIGINT       NOT NULL DEFAULT 0 COMMENT '公司ID',
  `code`            VARCHAR(20)  NOT NULL COMMENT '交付单编号 DV-xxx',
  `order_id`        BIGINT       NOT NULL DEFAULT 0 COMMENT '订单ID',
  `customer_id`     BIGINT       NOT NULL DEFAULT 0 COMMENT '客户ID',
  `customer_name`   VARCHAR(50)  DEFAULT NULL COMMENT '客户姓名(快照)',
  `stage`           VARCHAR(20)  NOT NULL DEFAULT 'upload_pending' COMMENT '阶段 upload_pending-待上传样片 selecting-客户选片中 retouching-精修进行中 deliver_ready-待确认交付 completed-已交付',
  `sample_count`    INT          NOT NULL DEFAULT 0 COMMENT '样片数量',
  `selected_count`  INT          NOT NULL DEFAULT 0 COMMENT '客户已选张数',
  `retouched_count` INT          NOT NULL DEFAULT 0 COMMENT '精修完成张数',
  `selected_at`     DATETIME     DEFAULT NULL COMMENT '选片完成时间',
  `delivered_at`    DATETIME     DEFAULT NULL COMMENT '交付时间',
  `remark`          VARCHAR(500) DEFAULT NULL COMMENT '备注',
  `operator_id`     BIGINT       NOT NULL DEFAULT 0 COMMENT '当前处理人ID',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_delivery_code` (`code`, `deleted`),
  KEY `idx_delivery_order` (`order_id`),
  KEY `idx_delivery_company` (`company_id`),
  KEY `idx_delivery_customer` (`customer_id`),
  KEY `idx_delivery_stage` (`stage`),
  KEY `idx_delivery_deleted` (`deleted`)
) ENGINE=InnoDB AUTO_INCREMENT=1 DEFAULT CHARSET=utf8 COLLATE=utf8_general_ci COMMENT='交付单';

-- 交付明细（样片/精修文件）
DROP TABLE IF EXISTS `biz_delivery_item`;
CREATE TABLE `biz_delivery_item` (
  `id`            BIGINT       NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `created_by`    BIGINT       NOT NULL DEFAULT 0 COMMENT '创建人',
  `created_at`    DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `updated_by`    BIGINT       NOT NULL DEFAULT 0 COMMENT '修改人',
  `updated_at`    DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '修改时间',
  `deleted`       BIGINT       NOT NULL DEFAULT 0 COMMENT '是否删除 0-否 1-是',
  `company_id`    BIGINT       NOT NULL DEFAULT 0 COMMENT '公司ID',
  `delivery_id`   BIGINT       NOT NULL DEFAULT 0 COMMENT '交付单ID',
  `order_id`      BIGINT       NOT NULL DEFAULT 0 COMMENT '订单ID',
  `url`           VARCHAR(500) NOT NULL COMMENT '文件地址',
  `file_type`     VARCHAR(10)  DEFAULT NULL COMMENT '类型 image-图片 video-视频',
  `kind`          VARCHAR(20)  DEFAULT NULL COMMENT '用途 sample-样片 selected-已选 retouched-精修成品',
  `filename`      VARCHAR(200) DEFAULT NULL COMMENT '原始文件名',
  `size`          BIGINT       DEFAULT NULL COMMENT '文件大小(字节)',
  PRIMARY KEY (`id`),
  KEY `idx_delivery_item_delivery` (`delivery_id`),
  KEY `idx_delivery_item_order` (`order_id`),
  KEY `idx_delivery_item_company` (`company_id`),
  KEY `idx_delivery_item_deleted` (`deleted`)
) ENGINE=InnoDB AUTO_INCREMENT=1 DEFAULT CHARSET=utf8 COLLATE=utf8_general_ci COMMENT='交付明细';

-- 作品集
DROP TABLE IF EXISTS `biz_asset`;
CREATE TABLE `biz_asset` (
  `id`            BIGINT        NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `created_by`    BIGINT        NOT NULL DEFAULT 0 COMMENT '创建人',
  `created_at`    DATETIME      NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `updated_by`    BIGINT        NOT NULL DEFAULT 0 COMMENT '修改人',
  `updated_at`    DATETIME      NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '修改时间',
  `deleted`       BIGINT        NOT NULL DEFAULT 0 COMMENT '是否删除 0-否 1-是',
  `company_id`    BIGINT        NOT NULL DEFAULT 0 COMMENT '公司ID',
  `code`          VARCHAR(20)   NOT NULL COMMENT '作品编号 WK-xxx',
  `title`         VARCHAR(100)  DEFAULT NULL COMMENT '作品标题',
  `category`      VARCHAR(50)   DEFAULT NULL COMMENT '作品类型(婚纱/写真/儿童/全家福/活动跟拍等)',
  `cover`         VARCHAR(500)  DEFAULT NULL COMMENT '封面图',
  `images`        TEXT          COMMENT '作品图片(逗号分隔)',
  `description`   VARCHAR(1000) DEFAULT NULL COMMENT '作品描述',
  `photographer`  VARCHAR(50)   DEFAULT NULL COMMENT '摄影师',
  `model`         VARCHAR(50)   DEFAULT NULL COMMENT '模特',
  `location`      VARCHAR(100)  DEFAULT NULL COMMENT '拍摄地点',
  `status`        VARCHAR(20)   NOT NULL DEFAULT 'draft' COMMENT '状态 draft-草稿 published-已发布',
  `view_count`    BIGINT        NOT NULL DEFAULT 0 COMMENT '浏览数',
  `published_at`  DATETIME      DEFAULT NULL COMMENT '发布时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_asset_code` (`code`, `deleted`),
  KEY `idx_asset_company` (`company_id`),
  KEY `idx_asset_category` (`category`),
  KEY `idx_asset_status` (`status`),
  KEY `idx_asset_deleted` (`deleted`)
) ENGINE=InnoDB AUTO_INCREMENT=1 DEFAULT CHARSET=utf8 COLLATE=utf8_general_ci COMMENT='作品集';

-- 档期锁定
DROP TABLE IF EXISTS `biz_calendar_block`;
CREATE TABLE `biz_calendar_block` (
  `id`             BIGINT       NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `created_by`     BIGINT       NOT NULL DEFAULT 0 COMMENT '创建人',
  `created_at`     DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `updated_by`     BIGINT       NOT NULL DEFAULT 0 COMMENT '修改人',
  `updated_at`     DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '修改时间',
  `deleted`        BIGINT       NOT NULL DEFAULT 0 COMMENT '是否删除 0-否 1-是',
  `company_id`     BIGINT       NOT NULL DEFAULT 0 COMMENT '公司ID',
  `store_id`       BIGINT       NOT NULL DEFAULT 0 COMMENT '所属门店ID',
  `order_id`       BIGINT       NOT NULL DEFAULT 0 COMMENT '关联订单ID',
  `customer_id`    BIGINT       NOT NULL DEFAULT 0 COMMENT '关联客户ID',
  `customer_name`  VARCHAR(50)  DEFAULT NULL COMMENT '客户姓名(快照)',
  `date`           VARCHAR(10)  DEFAULT NULL COMMENT '拍摄日期 yyyy-MM-dd',
  `time_range`     VARCHAR(50)  DEFAULT NULL COMMENT '时间段',
  `project_type`   VARCHAR(50)  DEFAULT NULL COMMENT '项目类型',
  `photographer_id` BIGINT      NOT NULL DEFAULT 0 COMMENT '摄影师ID',
  `photographer`   VARCHAR(50)  DEFAULT NULL COMMENT '摄影师姓名',
  `status`         VARCHAR(20)  NOT NULL DEFAULT 'locked' COMMENT '状态 locked-已锁定 cancelled-已取消',
  `remark`         VARCHAR(200) DEFAULT NULL COMMENT '备注',
  PRIMARY KEY (`id`),
  KEY `idx_block_company` (`company_id`),
  KEY `idx_block_store` (`store_id`),
  KEY `idx_block_order` (`order_id`),
  KEY `idx_block_date` (`date`),
  KEY `idx_block_photographer` (`photographer_id`),
  KEY `idx_block_deleted` (`deleted`)
) ENGINE=InnoDB AUTO_INCREMENT=1 DEFAULT CHARSET=utf8 COLLATE=utf8_general_ci COMMENT='档期锁定';

-- 收款方式
DROP TABLE IF EXISTS `biz_payment_method`;
CREATE TABLE `biz_payment_method` (
  `id`            BIGINT       NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `created_by`    BIGINT       NOT NULL DEFAULT 0 COMMENT '创建人',
  `created_at`    DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `updated_by`    BIGINT       NOT NULL DEFAULT 0 COMMENT '修改人',
  `updated_at`    DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '修改时间',
  `deleted`       BIGINT       NOT NULL DEFAULT 0 COMMENT '是否删除 0-否 1-是',
  `company_id`    BIGINT       NOT NULL DEFAULT 0 COMMENT '公司ID',
  `name`          VARCHAR(50)  NOT NULL COMMENT '收款方式名称',
  `type`          VARCHAR(20)  DEFAULT NULL COMMENT '类型 wechat-微信 alipay-支付宝 bank-银行转账 cash-现金 other-其他',
  `account_name`  VARCHAR(100) DEFAULT NULL COMMENT '收款账户名称',
  `account_no`    VARCHAR(100) DEFAULT NULL COMMENT '收款账号',
  `qrcode`        VARCHAR(500) DEFAULT NULL COMMENT '收款二维码',
  `status`        TINYINT      NOT NULL DEFAULT 1 COMMENT '状态 1-启用 0-停用',
  `sort`          INT          NOT NULL DEFAULT 0 COMMENT '排序',
  PRIMARY KEY (`id`),
  KEY `idx_paymethod_company` (`company_id`),
  KEY `idx_paymethod_deleted` (`deleted`)
) ENGINE=InnoDB AUTO_INCREMENT=1 DEFAULT CHARSET=utf8 COLLATE=utf8_general_ci COMMENT='收款方式';

-- 上传文件记录
DROP TABLE IF EXISTS `biz_upload`;
CREATE TABLE `biz_upload` (
  `id`            BIGINT       NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `created_by`    BIGINT       NOT NULL DEFAULT 0 COMMENT '创建人',
  `created_at`    DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `updated_by`    BIGINT       NOT NULL DEFAULT 0 COMMENT '修改人',
  `updated_at`    DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '修改时间',
  `deleted`       BIGINT       NOT NULL DEFAULT 0 COMMENT '是否删除 0-否 1-是',
  `company_id`    BIGINT       NOT NULL DEFAULT 0 COMMENT '公司ID',
  `store_id`      BIGINT       NOT NULL DEFAULT 0 COMMENT '所属门店ID',
  `biz_type`      VARCHAR(50)  DEFAULT NULL COMMENT '业务类型(订单/交付/作品等)',
  `biz_id`        BIGINT       NOT NULL DEFAULT 0 COMMENT '业务ID',
  `file_type`     VARCHAR(10)  DEFAULT NULL COMMENT '文件类型 image-图片 video-视频 file-文件',
  `file_name`     VARCHAR(200) DEFAULT NULL COMMENT '原始文件名',
  `file_url`      VARCHAR(500) NOT NULL COMMENT '文件访问地址',
  `file_path`     VARCHAR(500) DEFAULT NULL COMMENT '服务器存储路径',
  `size`          BIGINT       DEFAULT NULL COMMENT '文件大小(字节)',
  `upload_by`     BIGINT       NOT NULL DEFAULT 0 COMMENT '上传人ID',
  PRIMARY KEY (`id`),
  KEY `idx_upload_company` (`company_id`),
  KEY `idx_upload_store` (`store_id`),
  KEY `idx_upload_biz` (`biz_type`, `biz_id`),
  KEY `idx_upload_deleted` (`deleted`)
) ENGINE=InnoDB AUTO_INCREMENT=1 DEFAULT CHARSET=utf8 COLLATE=utf8_general_ci COMMENT='上传文件记录';
