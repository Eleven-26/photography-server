-- =====================================================================
-- SLOT 摄影工作室管理系统 数据库初始化脚本 (DDL · 全量)
-- 库名：photography   字符集：utf8   排序规则：utf8_general_ci
-- 用途：新建环境 / 新租户部署时一次性初始化（29 张表的最新结构）
--
-- 【维护约定 · 重要】
--   1) 本文件是「全量基线」。任何结构变更（建表/加字段/改索引）都必须：
--        a. 先写增量脚本到 docs/sql/增量/YYYYMMDD_描述.sql
--        b. 再把同样的改动合并进本文件（保持本文件始终是最新结构）
--   2) 一致性要求：本文件单独执行一次的结果，必须与
--      「增量/ 下所有脚本按日期顺序依次执行」的结果完全一致。
--      每次合并后都要实际验证（建两个临时库分别执行并比对 information_schema）。
--
-- 【通用约定】
--   1) 每张业务表固定包含：created_by 创建人 / created_at 创建时间 /
--      updated_by 修改人 / updated_at 修改时间 / deleted 是否删除(软删除)
--   2) 每张业务表均含 company_id，用于 SaaS 多租户(一公司多门店、多门店多管理员)
--   3) 所有字段均带中文注释
-- =====================================================================

CREATE DATABASE IF NOT EXISTS `photography` DEFAULT CHARACTER SET utf8 COLLATE utf8_general_ci;
USE `photography`;

SET NAMES utf8mb4;

-- ---------------------------------------------------------------------
-- 一、系统表（租户 / 门店 / 权限 / 设备 / 日志 / 通知）
-- ---------------------------------------------------------------------

-- 公司/工作室
DROP TABLE IF EXISTS `sys_company`;
CREATE TABLE `sys_company`
(
    `id`            bigint       NOT NULL AUTO_INCREMENT COMMENT '主键ID',
    `created_by`    bigint       NOT NULL DEFAULT '0' COMMENT '创建人',
    `created_at`    datetime     NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `updated_by`    bigint       NOT NULL DEFAULT '0' COMMENT '修改人',
    `updated_at`    datetime     NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '修改时间',
    `deleted`       bigint       NOT NULL DEFAULT '0' COMMENT '是否删除 0-否 1-是',
    `name`          varchar(100) NOT NULL COMMENT '公司/工作室名称',
    `logo`          varchar(500)          DEFAULT NULL COMMENT 'LOGO地址',
    `city`          varchar(50)           DEFAULT NULL COMMENT '所在城市',
    `intro`         varchar(1000)         DEFAULT NULL COMMENT '工作室简介',
    `contact_name`  varchar(50)           DEFAULT NULL COMMENT '联系人',
    `contact_phone` varchar(20)           DEFAULT NULL COMMENT '联系电话',
    `address`       varchar(200)          DEFAULT NULL COMMENT '地址',
    `status`        tinyint      NOT NULL DEFAULT '1' COMMENT '状态 1-正常 0-停用',
    PRIMARY KEY (`id`),
    KEY             `idx_company_deleted` (`deleted`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8 COMMENT='公司/工作室';

-- 门店
DROP TABLE IF EXISTS `sys_store`;
CREATE TABLE `sys_store`
(
    `id`         bigint       NOT NULL AUTO_INCREMENT COMMENT '主键ID',
    `created_by` bigint       NOT NULL DEFAULT '0' COMMENT '创建人',
    `created_at` datetime     NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `updated_by` bigint       NOT NULL DEFAULT '0' COMMENT '修改人',
    `updated_at` datetime     NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '修改时间',
    `deleted`    bigint       NOT NULL DEFAULT '0' COMMENT '是否删除 0-否 1-是',
    `company_id` bigint       NOT NULL DEFAULT '0' COMMENT '公司ID',
    `name`       varchar(100) NOT NULL COMMENT '门店名称',
    `address`    varchar(200)          DEFAULT NULL COMMENT '门店地址',
    `phone`      varchar(20)           DEFAULT NULL COMMENT '门店电话',
    `status`     tinyint      NOT NULL DEFAULT '1' COMMENT '状态 1-正常 0-停用',
    PRIMARY KEY (`id`),
    KEY          `idx_store_company` (`company_id`),
    KEY          `idx_store_deleted` (`deleted`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8 COMMENT='门店';

-- 角色
DROP TABLE IF EXISTS `sys_role`;
CREATE TABLE `sys_role`
(
    `id`         bigint      NOT NULL AUTO_INCREMENT COMMENT '主键ID',
    `created_by` bigint      NOT NULL DEFAULT '0' COMMENT '创建人',
    `created_at` datetime    NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `updated_by` bigint      NOT NULL DEFAULT '0' COMMENT '修改人',
    `updated_at` datetime    NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '修改时间',
    `deleted`    bigint      NOT NULL DEFAULT '0' COMMENT '是否删除 0-否 1-是',
    `company_id` bigint      NOT NULL DEFAULT '0' COMMENT '公司ID',
    `name`       varchar(50) NOT NULL COMMENT '角色名称',
    `code`       varchar(50) NOT NULL COMMENT '角色编码 admin-超级管理员 manager-店长 photographer-摄影师 sales-销售',
    `remark`     varchar(200)         DEFAULT NULL COMMENT '备注',
    `status`     tinyint     NOT NULL DEFAULT '1' COMMENT '状态 1-启用 0-停用',
    PRIMARY KEY (`id`),
    KEY          `idx_role_company` (`company_id`),
    KEY          `idx_role_deleted` (`deleted`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8 COMMENT='角色';

-- 后台管理员/员工
DROP TABLE IF EXISTS `sys_user`;
CREATE TABLE `sys_user`
(
    `id`            bigint       NOT NULL AUTO_INCREMENT COMMENT '主键ID',
    `created_by`    bigint       NOT NULL DEFAULT '0' COMMENT '创建人',
    `created_at`    datetime     NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `updated_by`    bigint       NOT NULL DEFAULT '0' COMMENT '修改人',
    `updated_at`    datetime     NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '修改时间',
    `deleted`       bigint       NOT NULL DEFAULT '0' COMMENT '是否删除 0-否 1-是',
    `company_id`    bigint       NOT NULL DEFAULT '0' COMMENT '公司ID',
    `store_id`      bigint       NOT NULL DEFAULT '0' COMMENT '所属门店ID',
    `username`      varchar(50)  NOT NULL COMMENT '登录账号',
    `password`      varchar(255) NOT NULL COMMENT '登录密码(bcrypt)',
    `nickname`      varchar(50)           DEFAULT NULL COMMENT '姓名/昵称',
    `mobile`        varchar(20)           DEFAULT NULL COMMENT '手机号',
    `email`         varchar(100)          DEFAULT NULL COMMENT '邮箱',
    `avatar`        varchar(500)          DEFAULT NULL COMMENT '头像地址',
    `role_id`       bigint       NOT NULL DEFAULT '0' COMMENT '角色ID',
    `status`        tinyint      NOT NULL DEFAULT '1' COMMENT '状态 1-启用 0-停用',
    `last_login_at` datetime              DEFAULT NULL COMMENT '最近登录时间',
    `last_login_ip` varchar(50)           DEFAULT NULL COMMENT '最近登录IP',
    PRIMARY KEY (`id`),
    UNIQUE KEY `uk_user_username` (`username`,`deleted`),
    KEY             `idx_user_company` (`company_id`),
    KEY             `idx_user_store` (`store_id`),
    KEY             `idx_user_role` (`role_id`),
    KEY             `idx_user_deleted` (`deleted`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8 COMMENT='后台管理员/员工';

-- 登录设备（摄影师 App 账号安全）
DROP TABLE IF EXISTS `sys_user_device`;
CREATE TABLE `sys_user_device`
(
    `id`             bigint   NOT NULL AUTO_INCREMENT COMMENT '主键ID',
    `created_by`     bigint   NOT NULL DEFAULT '0' COMMENT '创建人',
    `created_at`     datetime NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `updated_by`     bigint   NOT NULL DEFAULT '0' COMMENT '修改人',
    `updated_at`     datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '修改时间',
    `deleted`        bigint   NOT NULL DEFAULT '0' COMMENT '是否删除 0-否 1-是',
    `company_id`     bigint   NOT NULL DEFAULT '0' COMMENT '公司ID',
    `user_id`        bigint   NOT NULL DEFAULT '0' COMMENT '用户ID',
    `device_name`    varchar(100)      DEFAULT NULL COMMENT '设备名称(如 iPhone 15)',
    `platform`       varchar(20)       DEFAULT NULL COMMENT '平台 ios/android/pc/wechat',
    `last_ip`        varchar(50)       DEFAULT NULL COMMENT '最近登录IP',
    `last_active_at` datetime          DEFAULT NULL COMMENT '最近活跃时间',
    `status`         tinyint  NOT NULL DEFAULT '1' COMMENT '状态 1-正常 0-已踢出',
    PRIMARY KEY (`id`),
    KEY              `idx_device_user` (`user_id`),
    KEY              `idx_device_company` (`company_id`),
    KEY              `idx_device_deleted` (`deleted`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8 COMMENT='登录设备';

-- 操作日志
DROP TABLE IF EXISTS `sys_operation_log`;
CREATE TABLE `sys_operation_log`
(
    `id`         bigint   NOT NULL AUTO_INCREMENT COMMENT '主键ID',
    `created_by` bigint   NOT NULL DEFAULT '0' COMMENT '创建人',
    `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `updated_by` bigint   NOT NULL DEFAULT '0' COMMENT '修改人',
    `updated_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '修改时间',
    `deleted`    bigint   NOT NULL DEFAULT '0' COMMENT '是否删除 0-否 1-是',
    `company_id` bigint   NOT NULL DEFAULT '0' COMMENT '公司ID',
    `user_id`    bigint   NOT NULL DEFAULT '0' COMMENT '操作人ID',
    `username`   varchar(50)       DEFAULT NULL COMMENT '操作人账号',
    `module`     varchar(50)       DEFAULT NULL COMMENT '模块',
    `action`     varchar(100)      DEFAULT NULL COMMENT '操作行为',
    `method`     varchar(10)       DEFAULT NULL COMMENT '请求方法',
    `path`       varchar(200)      DEFAULT NULL COMMENT '请求路径',
    `params`     text COMMENT '请求参数',
    `ip`         varchar(50)       DEFAULT NULL COMMENT '请求IP',
    `status`     tinyint           DEFAULT NULL COMMENT '状态 1-成功 0-失败',
    `duration`   bigint            DEFAULT NULL COMMENT '耗时(毫秒)',
    PRIMARY KEY (`id`),
    KEY          `idx_oplog_user` (`user_id`),
    KEY          `idx_oplog_company` (`company_id`),
    KEY          `idx_oplog_created` (`created_at`),
    KEY          `idx_oplog_deleted` (`deleted`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8 COMMENT='操作日志';

-- 站内通知
DROP TABLE IF EXISTS `sys_notification`;
CREATE TABLE `sys_notification`
(
    `id`            bigint   NOT NULL AUTO_INCREMENT COMMENT '主键ID',
    `created_by`    bigint   NOT NULL DEFAULT '0' COMMENT '创建人',
    `created_at`    datetime NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `updated_by`    bigint   NOT NULL DEFAULT '0' COMMENT '修改人',
    `updated_at`    datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '修改时间',
    `deleted`       bigint   NOT NULL DEFAULT '0' COMMENT '是否删除 0-否 1-是',
    `company_id`    bigint   NOT NULL DEFAULT '0' COMMENT '公司ID',
    `receiver_id`   bigint   NOT NULL DEFAULT '0' COMMENT '接收人ID',
    `receiver_type` tinyint  NOT NULL DEFAULT '1' COMMENT '接收人类型 1-员工 2-客户',
    `type`          tinyint           DEFAULT NULL COMMENT '类型 1-订单 2-财务 3-系统',
    `title`         varchar(100)      DEFAULT NULL COMMENT '标题',
    `content`       varchar(500)      DEFAULT NULL COMMENT '内容',
    `biz_type`      varchar(20)       DEFAULT NULL COMMENT '业务类型 order-订单 refund-退款',
    `biz_id`        bigint            DEFAULT NULL COMMENT '业务ID',
    `is_read`       tinyint  NOT NULL DEFAULT '0' COMMENT '是否已读 0-未读 1-已读',
    `read_at`       datetime          DEFAULT NULL COMMENT '已读时间',
    PRIMARY KEY (`id`),
    KEY             `idx_notice_receiver` (`receiver_id`),
    KEY             `idx_notice_company` (`company_id`),
    KEY             `idx_notice_read` (`is_read`),
    KEY             `idx_notice_deleted` (`deleted`),
    KEY             `idx_notice_receiver_type` (`receiver_type`,`receiver_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8 COMMENT='站内通知';

-- ---------------------------------------------------------------------
-- 二、客户与线索（CRM + 客户互动）
-- ---------------------------------------------------------------------

-- 客户
DROP TABLE IF EXISTS `crm_customer`;
CREATE TABLE `crm_customer`
(
    `id`           bigint         NOT NULL AUTO_INCREMENT COMMENT '主键ID',
    `created_by`   bigint         NOT NULL DEFAULT '0' COMMENT '创建人',
    `created_at`   datetime       NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `updated_by`   bigint         NOT NULL DEFAULT '0' COMMENT '修改人',
    `updated_at`   datetime       NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '修改时间',
    `deleted`      bigint         NOT NULL DEFAULT '0' COMMENT '是否删除 0-否 1-是',
    `company_id`   bigint         NOT NULL DEFAULT '0' COMMENT '公司ID',
    `code`         varchar(20)    NOT NULL COMMENT '客户编号 CU-xxx',
    `store_id`     bigint         NOT NULL DEFAULT '0' COMMENT '所属门店ID',
    `name`         varchar(50)    NOT NULL COMMENT '客户姓名',
    `mobile`       varchar(20)             DEFAULT NULL COMMENT '手机号',
    `wechat`       varchar(50)             DEFAULT NULL COMMENT '微信号',
    `gender`       varchar(10)             DEFAULT NULL COMMENT '性别 male-男 female-女 unknown-未知',
    `birthday`     varchar(20)             DEFAULT NULL COMMENT '生日',
    `level`        tinyint        NOT NULL DEFAULT '4' COMMENT '客户等级 1-钻石 2-铂金 3-黄金 4-普通',
    `source`       varchar(50)             DEFAULT NULL COMMENT '客户来源',
    `tags`         varchar(200)            DEFAULT NULL COMMENT '标签(逗号分隔)',
    `status`       tinyint        NOT NULL DEFAULT '1' COMMENT '状态 1-潜在 2-活跃 3-流失',
    `remark`       varchar(500)            DEFAULT NULL COMMENT '备注',
    `avatar`       varchar(500)            DEFAULT NULL COMMENT '头像地址',
    `openid`       varchar(64)             DEFAULT NULL COMMENT '微信小程序openid',
    `unionid`      varchar(64)             DEFAULT NULL COMMENT '微信unionid',
    `is_verified`  tinyint        NOT NULL DEFAULT '0' COMMENT '手机号是否已验证 0-否 1-是',
    `order_count`  bigint         NOT NULL DEFAULT '0' COMMENT '订单数(冗余)',
    `total_amount` decimal(12, 2) NOT NULL DEFAULT '0.00' COMMENT '累计消费(冗余)',
    PRIMARY KEY (`id`),
    UNIQUE KEY `uk_customer_code` (`code`,`deleted`),
    KEY            `idx_customer_mobile` (`mobile`),
    KEY            `idx_customer_company` (`company_id`),
    KEY            `idx_customer_store` (`store_id`),
    KEY            `idx_customer_deleted` (`deleted`),
    KEY            `idx_customer_openid` (`openid`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8 COMMENT='客户';

-- 线索
DROP TABLE IF EXISTS `crm_lead`;
CREATE TABLE `crm_lead`
(
    `id`             bigint      NOT NULL AUTO_INCREMENT COMMENT '主键ID',
    `created_by`     bigint      NOT NULL DEFAULT '0' COMMENT '创建人',
    `created_at`     datetime    NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `updated_by`     bigint      NOT NULL DEFAULT '0' COMMENT '修改人',
    `updated_at`     datetime    NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '修改时间',
    `deleted`        bigint      NOT NULL DEFAULT '0' COMMENT '是否删除 0-否 1-是',
    `company_id`     bigint      NOT NULL DEFAULT '0' COMMENT '公司ID',
    `code`           varchar(20) NOT NULL COMMENT '线索编号 LD-xxx',
    `store_id`       bigint      NOT NULL DEFAULT '0' COMMENT '所属门店ID',
    `customer_id`    bigint      NOT NULL DEFAULT '0' COMMENT '关联客户ID',
    `name`           varchar(50)          DEFAULT NULL COMMENT '客户姓名',
    `mobile`         varchar(20)          DEFAULT NULL COMMENT '手机号',
    `source`         varchar(50)          DEFAULT NULL COMMENT '线索来源',
    `project_type`   varchar(50)          DEFAULT NULL COMMENT '意向项目类型(婚纱/写真/儿童/全家福/活动跟拍等)',
    `budget_min`     decimal(12, 2)       DEFAULT NULL COMMENT '预算区间-下限',
    `budget_max`     decimal(12, 2)       DEFAULT NULL COMMENT '预算区间-上限',
    `status`         tinyint     NOT NULL DEFAULT '1' COMMENT '状态 1-待回复 2-待报价 3-已报价 4-已成交 5-已流失',
    `shoot_date`     varchar(20)          DEFAULT NULL COMMENT '意向拍摄日期',
    `remark`         varchar(500)         DEFAULT NULL COMMENT '备注',
    `owner_id`       bigint      NOT NULL DEFAULT '0' COMMENT '负责人ID',
    `next_follow_at` datetime             DEFAULT NULL COMMENT '下次跟进时间',
    `follower`       int         NOT NULL DEFAULT '0' COMMENT '跟进次数',
    `last_follow_at` datetime             DEFAULT NULL COMMENT '最近跟进时间',
    PRIMARY KEY (`id`),
    UNIQUE KEY `uk_lead_code` (`code`,`deleted`),
    KEY              `idx_lead_company` (`company_id`),
    KEY              `idx_lead_store` (`store_id`),
    KEY              `idx_lead_customer` (`customer_id`),
    KEY              `idx_lead_owner` (`owner_id`),
    KEY              `idx_lead_status` (`status`),
    KEY              `idx_lead_deleted` (`deleted`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8 COMMENT='线索';

-- 定制需求（客户非套餐需求提交 → 可转化线索）
DROP TABLE IF EXISTS `biz_custom_request`;
CREATE TABLE `biz_custom_request`
(
    `id`            bigint   NOT NULL AUTO_INCREMENT COMMENT '主键ID',
    `created_by`    bigint   NOT NULL DEFAULT '0' COMMENT '创建人',
    `created_at`    datetime NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `updated_by`    bigint   NOT NULL DEFAULT '0' COMMENT '修改人',
    `updated_at`    datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '修改时间',
    `deleted`       bigint   NOT NULL DEFAULT '0' COMMENT '是否删除 0-否 1-是',
    `company_id`    bigint   NOT NULL DEFAULT '0' COMMENT '公司ID',
    `customer_id`   bigint   NOT NULL DEFAULT '0' COMMENT '客户ID(登录提交时有值)',
    `name`          varchar(50)       DEFAULT NULL COMMENT '称呼',
    `mobile`        varchar(20)       DEFAULT NULL COMMENT '联系电话',
    `project_type`  varchar(50)       DEFAULT NULL COMMENT '拍摄类型(家庭纪念/个人写真/情侣/婚纱/儿童写真/活动跟拍/其他)',
    `expected_date` varchar(50)       DEFAULT NULL COMMENT '期望拍摄日期(如 2026年8月下旬)',
    `location`      varchar(200)      DEFAULT NULL COMMENT '期望拍摄地点',
    `budget_min`    decimal(12, 2)    DEFAULT NULL COMMENT '预算下限',
    `budget_max`    decimal(12, 2)    DEFAULT NULL COMMENT '预算上限',
    `detail`        varchar(1000)     DEFAULT NULL COMMENT '详细需求',
    `images`        text COMMENT '参考图片(逗号分隔)',
    `status`        tinyint  NOT NULL DEFAULT '1' COMMENT '状态 1-待处理 2-已响应 3-已关闭',
    `lead_id`       bigint   NOT NULL DEFAULT '0' COMMENT '转化线索ID',
    `response`      varchar(500)      DEFAULT NULL COMMENT '响应说明',
    `response_by`   bigint   NOT NULL DEFAULT '0' COMMENT '响应人ID',
    `response_at`   datetime          DEFAULT NULL COMMENT '响应时间',
    PRIMARY KEY (`id`),
    KEY             `idx_custom_req_company` (`company_id`),
    KEY             `idx_custom_req_customer` (`customer_id`),
    KEY             `idx_custom_req_status` (`status`),
    KEY             `idx_custom_req_deleted` (`deleted`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8 COMMENT='定制需求';

-- 线索沟通记录（客户 H5 咨询/摄影师追问/发送作品等往来消息）
DROP TABLE IF EXISTS `biz_lead_message`;
CREATE TABLE `biz_lead_message`
(
    `id`          bigint   NOT NULL AUTO_INCREMENT COMMENT '主键ID',
    `created_by`  bigint   NOT NULL DEFAULT '0' COMMENT '创建人',
    `created_at`  datetime NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `updated_by`  bigint   NOT NULL DEFAULT '0' COMMENT '修改人',
    `updated_at`  datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '修改时间',
    `deleted`     bigint   NOT NULL DEFAULT '0' COMMENT '是否删除 0-否 1-是',
    `company_id`  bigint   NOT NULL DEFAULT '0' COMMENT '公司ID',
    `lead_id`     bigint   NOT NULL DEFAULT '0' COMMENT '线索ID',
    `customer_id` bigint   NOT NULL DEFAULT '0' COMMENT '客户ID',
    `direction`   tinyint  NOT NULL DEFAULT '1' COMMENT '方向 1-客户发来 2-工作室发出',
    `channel`     varchar(20)       DEFAULT NULL COMMENT '渠道 h5-预约页 wechat-微信 sms-短信 phone-电话',
    `content`     varchar(1000)     DEFAULT NULL COMMENT '消息内容',
    `msg_type`    tinyint  NOT NULL DEFAULT '1' COMMENT '类型 1-文本 2-追问 3-报价通知 4-作品分享',
    `biz_id`      bigint   NOT NULL DEFAULT '0' COMMENT '关联业务ID(报价单ID/作品ID等)',
    PRIMARY KEY (`id`),
    KEY           `idx_leadmsg_lead` (`lead_id`),
    KEY           `idx_leadmsg_company` (`company_id`),
    KEY           `idx_leadmsg_deleted` (`deleted`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8 COMMENT='线索沟通记录';

-- AI 简报项（从客户沟通中提取/待追问的需求项）
DROP TABLE IF EXISTS `biz_lead_brief_item`;
CREATE TABLE `biz_lead_brief_item`
(
    `id`              bigint   NOT NULL AUTO_INCREMENT COMMENT '主键ID',
    `created_by`      bigint   NOT NULL DEFAULT '0' COMMENT '创建人',
    `created_at`      datetime NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `updated_by`      bigint   NOT NULL DEFAULT '0' COMMENT '修改人',
    `updated_at`      datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '修改时间',
    `deleted`         bigint   NOT NULL DEFAULT '0' COMMENT '是否删除 0-否 1-是',
    `company_id`      bigint   NOT NULL DEFAULT '0' COMMENT '公司ID',
    `lead_id`         bigint   NOT NULL DEFAULT '0' COMMENT '线索ID',
    `title`           varchar(100)      DEFAULT NULL COMMENT '信息项标题(具体日期/儿童作息/妆容需求等)',
    `value`           varchar(500)      DEFAULT NULL COMMENT '已确认的值(已确认项)',
    `question`        varchar(500)      DEFAULT NULL COMMENT '待追问问题',
    `ai_suggestion`   varchar(500)      DEFAULT NULL COMMENT 'AI 建议追问话术',
    `status`          tinyint  NOT NULL DEFAULT '1' COMMENT '状态 1-待追问 2-已发送 3-已确认',
    `affects_pricing` tinyint  NOT NULL DEFAULT '0' COMMENT '是否影响报价 0-否 1-是',
    `sort`            int      NOT NULL DEFAULT '0' COMMENT '排序(按影响报价优先)',
    `sent_at`         datetime          DEFAULT NULL COMMENT '追问发送时间',
    `confirmed_at`    datetime          DEFAULT NULL COMMENT '确认时间',
    PRIMARY KEY (`id`),
    KEY               `idx_briefitem_lead` (`lead_id`),
    KEY               `idx_briefitem_company` (`company_id`),
    KEY               `idx_briefitem_status` (`status`),
    KEY               `idx_briefitem_deleted` (`deleted`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8 COMMENT='线索AI简报项';

-- ---------------------------------------------------------------------
-- 三、套餐与报价
-- ---------------------------------------------------------------------

-- 报价单
DROP TABLE IF EXISTS `biz_quote`;
CREATE TABLE `biz_quote`
(
    `id`             bigint         NOT NULL AUTO_INCREMENT COMMENT '主键ID',
    `created_by`     bigint         NOT NULL DEFAULT '0' COMMENT '创建人',
    `created_at`     datetime       NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `updated_by`     bigint         NOT NULL DEFAULT '0' COMMENT '修改人',
    `updated_at`     datetime       NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '修改时间',
    `deleted`        bigint         NOT NULL DEFAULT '0' COMMENT '是否删除 0-否 1-是',
    `company_id`     bigint         NOT NULL DEFAULT '0' COMMENT '公司ID',
    `code`           varchar(20)    NOT NULL COMMENT '报价单编号 QT-xxx',
    `lead_id`        bigint         NOT NULL DEFAULT '0' COMMENT '关联线索ID',
    `customer_id`    bigint         NOT NULL DEFAULT '0' COMMENT '关联客户ID',
    `package_id`     bigint         NOT NULL DEFAULT '0' COMMENT '关联套餐ID',
    `version`        int            NOT NULL DEFAULT '1' COMMENT '套餐版本号',
    `title`          varchar(100)            DEFAULT NULL COMMENT '报价标题',
    `package_name`   varchar(100)            DEFAULT NULL COMMENT '套餐名称(下单时快照)',
    `base_price`     decimal(12, 2)          DEFAULT NULL COMMENT '基础套餐价(快照)',
    `addon_price`    decimal(12, 2) NOT NULL DEFAULT '0.00' COMMENT '加选金额(快照)',
    `total_price`    decimal(12, 2)          DEFAULT NULL COMMENT '报价总额',
    `status`         tinyint        NOT NULL DEFAULT '1' COMMENT '状态 1-草稿 2-已发送 3-已接受 4-已拒绝 5-已成交',
    `valid_until`    datetime                DEFAULT NULL COMMENT '报价有效期至',
    `remark`         varchar(500)            DEFAULT NULL COMMENT '备注',
    `owner_id`       bigint         NOT NULL DEFAULT '0' COMMENT '负责人ID',
    `shoot_date`     varchar(20)             DEFAULT NULL COMMENT '意向拍摄日期',
    `shoot_time`     varchar(20)             DEFAULT NULL COMMENT '拍摄时间段',
    `duration_hours` decimal(5, 2)           DEFAULT NULL COMMENT '拍摄时长(小时)',
    `location`       varchar(200)            DEFAULT NULL COMMENT '拍摄地点',
    `location_type`  varchar(20)             DEFAULT NULL COMMENT '地点类型 outdoor-外拍 studio-合作影棚',
    `location_fee`   decimal(12, 2) NOT NULL DEFAULT '0.00' COMMENT '场地费',
    `people_count`   varchar(20)             DEFAULT NULL COMMENT '拍摄人数',
    `addons`         text COMMENT '加项清单(JSON数组: [{name,price,qty}])',
    `accept_at`      datetime                DEFAULT NULL COMMENT '客户接受时间',
    `order_id`       bigint         NOT NULL DEFAULT '0' COMMENT '转化生成的订单ID',
    PRIMARY KEY (`id`),
    UNIQUE KEY `uk_quote_code` (`code`,`deleted`),
    KEY              `idx_quote_company` (`company_id`),
    KEY              `idx_quote_lead` (`lead_id`),
    KEY              `idx_quote_customer` (`customer_id`),
    KEY              `idx_quote_package` (`package_id`),
    KEY              `idx_quote_deleted` (`deleted`),
    KEY              `idx_quote_order` (`order_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8 COMMENT='报价单';

-- 套餐（版本化管理：被订单引用后改价需生成新版本）
DROP TABLE IF EXISTS `biz_package`;
CREATE TABLE `biz_package`
(
    `id`                bigint         NOT NULL AUTO_INCREMENT COMMENT '主键ID',
    `created_by`        bigint         NOT NULL DEFAULT '0' COMMENT '创建人',
    `created_at`        datetime       NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `updated_by`        bigint         NOT NULL DEFAULT '0' COMMENT '修改人',
    `updated_at`        datetime       NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '修改时间',
    `deleted`           bigint         NOT NULL DEFAULT '0' COMMENT '是否删除 0-否 1-是',
    `company_id`        bigint         NOT NULL DEFAULT '0' COMMENT '公司ID',
    `code`              varchar(20)    NOT NULL COMMENT '套餐编号 PK-xxx',
    `store_id`          bigint         NOT NULL DEFAULT '0' COMMENT '所属门店ID',
    `name`              varchar(100)   NOT NULL COMMENT '套餐名称',
    `cover`             varchar(500)            DEFAULT NULL COMMENT '套餐封面图',
    `category`          varchar(50)             DEFAULT NULL COMMENT '套餐类型(婚纱/写真/儿童/全家福/活动跟拍等)',
    `suitable_for`      varchar(100)            DEFAULT NULL COMMENT '适合人群(逗号分隔: 家庭/亲子/纪念日/孕妇)',
    `introduction`      varchar(500)            DEFAULT NULL COMMENT '套餐简介(一句话)',
    `base_price`        decimal(12, 2)          DEFAULT NULL COMMENT '基础套餐价',
    `deposit_rate`      decimal(5, 2)  NOT NULL DEFAULT '30.00' COMMENT '定金比例(%)',
    `deposit_amt`       decimal(12, 2)          DEFAULT NULL COMMENT '定金金额(基础价×比例)',
    `photos_included`   int            NOT NULL DEFAULT '0' COMMENT '包含精修张数',
    `raw_count`         int            NOT NULL DEFAULT '0' COMMENT '原片数量(如 100+)',
    `revision_count`    int            NOT NULL DEFAULT '2' COMMENT '修改次数',
    `delivery_days`     int            NOT NULL DEFAULT '7' COMMENT '交付周期(工作日)',
    `download_days`     int            NOT NULL DEFAULT '30' COMMENT '成片下载有效期(天)',
    `location_mode`     varchar(20)    NOT NULL DEFAULT 'custom' COMMENT '地点模式 fixed-固定地点 custom-客户可选',
    `locations`         text COMMENT '可选地点(JSON数组: [{name,extra_fee,is_default}])',
    `reschedule_policy` varchar(200)            DEFAULT NULL COMMENT '改期政策文案',
    `cancel_policy`     varchar(200)            DEFAULT NULL COMMENT '取消政策文案',
    `shoot_hours`       decimal(5, 2)  NOT NULL DEFAULT '0.00' COMMENT '拍摄时长(小时)',
    `content_desc`      text COMMENT '包含内容说明',
    `addon_unit_price`  decimal(12, 2) NOT NULL DEFAULT '0.00' COMMENT '加选精修单价',
    `status`            tinyint        NOT NULL DEFAULT '2' COMMENT '状态 1-已上架 2-草稿 3-已下线',
    `version`           int            NOT NULL DEFAULT '1' COMMENT '套餐版本号',
    `base_version`      int            NOT NULL DEFAULT '0' COMMENT '上一版本号(0表示初始版本)',
    `published_at`      datetime                DEFAULT NULL COMMENT '上架时间',
    PRIMARY KEY (`id`),
    UNIQUE KEY `uk_package_code` (`code`,`deleted`),
    KEY                 `idx_package_company` (`company_id`),
    KEY                 `idx_package_store` (`store_id`),
    KEY                 `idx_package_status` (`status`),
    KEY                 `idx_package_deleted` (`deleted`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8 COMMENT='套餐';

-- ---------------------------------------------------------------------
-- 四、订单（主流程 + 收款/退款/加项/改期/评价/日志）
-- ---------------------------------------------------------------------

-- 订单
DROP TABLE IF EXISTS `biz_order`;
CREATE TABLE `biz_order`
(
    `id`                 bigint         NOT NULL AUTO_INCREMENT COMMENT '主键ID',
    `created_by`         bigint         NOT NULL DEFAULT '0' COMMENT '创建人',
    `created_at`         datetime       NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `updated_by`         bigint         NOT NULL DEFAULT '0' COMMENT '修改人',
    `updated_at`         datetime       NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '修改时间',
    `deleted`            bigint         NOT NULL DEFAULT '0' COMMENT '是否删除 0-否 1-是',
    `company_id`         bigint         NOT NULL DEFAULT '0' COMMENT '公司ID',
    `code`               varchar(30)    NOT NULL COMMENT '订单编号 SL-xxx',
    `store_id`           bigint         NOT NULL DEFAULT '0' COMMENT '所属门店ID',
    `customer_id`        bigint         NOT NULL DEFAULT '0' COMMENT '客户ID',
    `customer_name`      varchar(50)             DEFAULT NULL COMMENT '客户姓名(快照)',
    `customer_mobile`    varchar(20)             DEFAULT NULL COMMENT '客户手机号(快照)',
    `lead_id`            bigint         NOT NULL DEFAULT '0' COMMENT '来源线索ID',
    `quote_id`           bigint         NOT NULL DEFAULT '0' COMMENT '来源报价单ID',
    `package_id`         bigint         NOT NULL DEFAULT '0' COMMENT '套餐ID',
    `package_name`       varchar(100)            DEFAULT NULL COMMENT '套餐名称(快照)',
    `package_version`    int            NOT NULL DEFAULT '1' COMMENT '下单套餐版本号',
    `base_price`         decimal(12, 2)          DEFAULT NULL COMMENT '基础套餐价(快照)',
    `addon_amount`       decimal(12, 2) NOT NULL DEFAULT '0.00' COMMENT '加选精修金额(全部进尾款)',
    `deposit_amt`        decimal(12, 2) NOT NULL DEFAULT '0.00' COMMENT '定金金额(基础价×定金比例)',
    `final_amt`          decimal(12, 2) NOT NULL DEFAULT '0.00' COMMENT '尾款金额(基础价-定金+加选)',
    `total_amt`          decimal(12, 2) NOT NULL DEFAULT '0.00' COMMENT '订单总额',
    `paid_amt`           decimal(12, 2) NOT NULL DEFAULT '0.00' COMMENT '已收款金额',
    `refund_amt`         decimal(12, 2) NOT NULL DEFAULT '0.00' COMMENT '已退款金额',
    `status`             tinyint        NOT NULL DEFAULT '1' COMMENT '订单状态 0-待确认(客户预约) 1-待定金 2-待拍摄 3-拍摄中 4-精修中 5-待交付 6-已完成 7-已取消',
    `payment_status`     tinyint        NOT NULL DEFAULT '4' COMMENT '支付状态 1-待核验 2-已确认 3-已全额 4-待支付 5-已退款',
    `shoot_date`         varchar(20)             DEFAULT NULL COMMENT '拍摄日期',
    `shoot_time`         varchar(20)             DEFAULT NULL COMMENT '拍摄时间段',
    `shoot_address`      varchar(200)            DEFAULT NULL COMMENT '拍摄地点',
    `people_count`       varchar(20)             DEFAULT NULL COMMENT '拍摄人数(如 2大1小)',
    `shoot_style`        varchar(50)             DEFAULT NULL COMMENT '拍摄风格',
    `photographer_id`    bigint         NOT NULL DEFAULT '0' COMMENT '摄影师ID',
    `photographer`       varchar(50)             DEFAULT NULL COMMENT '摄影师姓名',
    `remark`             varchar(500)            DEFAULT NULL COMMENT '备注',
    `source_type`        tinyint        NOT NULL DEFAULT '1' COMMENT '订单来源 1-管理端录入 2-客户预约(H5/小程序) 3-线索报价转化',
    `prep_content`       text COMMENT '拍前准备清单内容(时间/地点/服装/交通/注意事项)',
    `prep_read_at`       datetime                DEFAULT NULL COMMENT '客户确认阅读拍前准备时间',
    `select_deadline`    datetime                DEFAULT NULL COMMENT '选片截止时间(逾期默认全选)',
    `delivery_expire_at` datetime                DEFAULT NULL COMMENT '成片下载有效期至',
    `is_archived`        tinyint        NOT NULL DEFAULT '0' COMMENT '是否已归档 0-否 1-是',
    `archived_at`        datetime                DEFAULT NULL COMMENT '归档时间',
    `cancel_reason`      varchar(200)            DEFAULT NULL COMMENT '取消原因',
    `finished_at`        datetime                DEFAULT NULL COMMENT '完成时间',
    `owner_id`           bigint         NOT NULL DEFAULT '0' COMMENT '负责人(销售)ID',
    PRIMARY KEY (`id`),
    UNIQUE KEY `uk_order_code` (`code`,`deleted`),
    KEY                  `idx_order_company` (`company_id`),
    KEY                  `idx_order_store` (`store_id`),
    KEY                  `idx_order_customer` (`customer_id`),
    KEY                  `idx_order_status` (`status`),
    KEY                  `idx_order_payment_status` (`payment_status`),
    KEY                  `idx_order_shoot_date` (`shoot_date`),
    KEY                  `idx_order_owner` (`owner_id`),
    KEY                  `idx_order_deleted` (`deleted`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8 COMMENT='订单';

-- 收款记录
DROP TABLE IF EXISTS `biz_order_payment`;
CREATE TABLE `biz_order_payment`
(
    `id`            bigint         NOT NULL AUTO_INCREMENT COMMENT '主键ID',
    `created_by`    bigint         NOT NULL DEFAULT '0' COMMENT '创建人',
    `created_at`    datetime       NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `updated_by`    bigint         NOT NULL DEFAULT '0' COMMENT '修改人',
    `updated_at`    datetime       NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '修改时间',
    `deleted`       bigint         NOT NULL DEFAULT '0' COMMENT '是否删除 0-否 1-是',
    `company_id`    bigint         NOT NULL DEFAULT '0' COMMENT '公司ID',
    `order_id`      bigint         NOT NULL DEFAULT '0' COMMENT '订单ID',
    `code`          varchar(20)    NOT NULL COMMENT '收款单号 PM-xxx',
    `customer_id`   bigint         NOT NULL DEFAULT '0' COMMENT '客户ID',
    `type`          varchar(20)             DEFAULT NULL COMMENT '收款类型 deposit-定金 final-尾款 addon-加选',
    `amount`        decimal(12, 2) NOT NULL COMMENT '收款金额',
    `method_id`     bigint         NOT NULL DEFAULT '0' COMMENT '收款方式ID',
    `method_name`   varchar(50)             DEFAULT NULL COMMENT '收款方式名称(快照)',
    `status`        tinyint        NOT NULL DEFAULT '1' COMMENT '状态 1-待核验 2-已确认 3-已退款',
    `paid_at`       datetime                DEFAULT NULL COMMENT '收款时间',
    `voucher`       varchar(500)            DEFAULT NULL COMMENT '收款凭证图片',
    `operator_id`   bigint         NOT NULL DEFAULT '0' COMMENT '收款操作人ID',
    `operator_name` varchar(50)             DEFAULT NULL COMMENT '收款操作人',
    `remark`        varchar(200)            DEFAULT NULL COMMENT '备注',
    PRIMARY KEY (`id`),
    UNIQUE KEY `uk_payment_code` (`code`,`deleted`),
    KEY             `idx_payment_order` (`order_id`),
    KEY             `idx_payment_company` (`company_id`),
    KEY             `idx_payment_customer` (`customer_id`),
    KEY             `idx_payment_status` (`status`),
    KEY             `idx_payment_deleted` (`deleted`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8 COMMENT='收款记录';

-- 退款记录
DROP TABLE IF EXISTS `biz_order_refund`;
CREATE TABLE `biz_order_refund`
(
    `id`                  bigint         NOT NULL AUTO_INCREMENT COMMENT '主键ID',
    `created_by`          bigint         NOT NULL DEFAULT '0' COMMENT '创建人',
    `created_at`          datetime       NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `updated_by`          bigint         NOT NULL DEFAULT '0' COMMENT '修改人',
    `updated_at`          datetime       NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '修改时间',
    `deleted`             bigint         NOT NULL DEFAULT '0' COMMENT '是否删除 0-否 1-是',
    `company_id`          bigint         NOT NULL DEFAULT '0' COMMENT '公司ID',
    `order_id`            bigint         NOT NULL DEFAULT '0' COMMENT '订单ID',
    `code`                varchar(20)    NOT NULL COMMENT '退款单号 RF-xxx',
    `customer_id`         bigint         NOT NULL DEFAULT '0' COMMENT '客户ID',
    `amount`              decimal(12, 2) NOT NULL COMMENT '退款金额',
    `reason`              varchar(200)            DEFAULT NULL COMMENT '退款原因',
    `reason_label`        varchar(50)             DEFAULT NULL COMMENT '取消原因分类(时间冲突/预算原因/找到其他摄影师/其他)',
    `apply_source`        tinyint        NOT NULL DEFAULT '1' COMMENT '申请来源 1-管理端 2-客户H5/小程序',
    `voucher`             varchar(500)            DEFAULT NULL COMMENT '退款转账凭证截图',
    `customer_confirm_at` datetime                DEFAULT NULL COMMENT '客户确认收到退款时间',
    `refund_rule`         varchar(50)             DEFAULT NULL COMMENT '退款规则档位(按拍摄前小时)',
    `status`              tinyint        NOT NULL DEFAULT '1' COMMENT '状态 1-申请中 2-已通过 3-已退款 4-已驳回',
    `apply_by`            bigint         NOT NULL DEFAULT '0' COMMENT '申请人ID',
    `apply_name`          varchar(50)             DEFAULT NULL COMMENT '申请人',
    `audit_by`            bigint         NOT NULL DEFAULT '0' COMMENT '审核人ID',
    `audit_at`            datetime                DEFAULT NULL COMMENT '审核时间',
    `audit_remark`        varchar(200)            DEFAULT NULL COMMENT '审核备注',
    `refund_at`           datetime                DEFAULT NULL COMMENT '退款时间',
    PRIMARY KEY (`id`),
    UNIQUE KEY `uk_refund_code` (`code`,`deleted`),
    KEY                   `idx_refund_order` (`order_id`),
    KEY                   `idx_refund_company` (`company_id`),
    KEY                   `idx_refund_customer` (`customer_id`),
    KEY                   `idx_refund_status` (`status`),
    KEY                   `idx_refund_deleted` (`deleted`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8 COMMENT='退款记录';

-- 订单加项（妆造/加急/收选等自定义加项；加选精修走 delivery.extra）
DROP TABLE IF EXISTS `biz_order_addon`;
CREATE TABLE `biz_order_addon`
(
    `id`         bigint         NOT NULL AUTO_INCREMENT COMMENT '主键ID',
    `created_by` bigint         NOT NULL DEFAULT '0' COMMENT '创建人',
    `created_at` datetime       NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `updated_by` bigint         NOT NULL DEFAULT '0' COMMENT '修改人',
    `updated_at` datetime       NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '修改时间',
    `deleted`    bigint         NOT NULL DEFAULT '0' COMMENT '是否删除 0-否 1-是',
    `company_id` bigint         NOT NULL DEFAULT '0' COMMENT '公司ID',
    `order_id`   bigint         NOT NULL DEFAULT '0' COMMENT '订单ID',
    `name`       varchar(100)   NOT NULL COMMENT '加项名称(妆造服务/加急交付/收选服务等)',
    `category`   varchar(20)             DEFAULT NULL COMMENT '分类 makeup-妆造 urgency-时效 service-服务 retouch-精修',
    `price`      decimal(12, 2) NOT NULL DEFAULT '0.00' COMMENT '单价',
    `qty`        int            NOT NULL DEFAULT '1' COMMENT '数量',
    `amount`     decimal(12, 2) NOT NULL DEFAULT '0.00' COMMENT '小计(price×qty)',
    `confirmed`  tinyint        NOT NULL DEFAULT '0' COMMENT '客户是否确认 0-待确认 1-已确认',
    `remark`     varchar(200)            DEFAULT NULL COMMENT '备注',
    PRIMARY KEY (`id`),
    KEY          `idx_addon_order` (`order_id`),
    KEY          `idx_addon_company` (`company_id`),
    KEY          `idx_addon_deleted` (`deleted`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8 COMMENT='订单加项';

-- 改期单（客户申请/摄影师发起 → 摄影师审批 → 档期变更）
DROP TABLE IF EXISTS `biz_order_reschedule`;
CREATE TABLE `biz_order_reschedule`
(
    `id`            bigint         NOT NULL AUTO_INCREMENT COMMENT '主键ID',
    `created_by`    bigint         NOT NULL DEFAULT '0' COMMENT '创建人',
    `created_at`    datetime       NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `updated_by`    bigint         NOT NULL DEFAULT '0' COMMENT '修改人',
    `updated_at`    datetime       NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '修改时间',
    `deleted`       bigint         NOT NULL DEFAULT '0' COMMENT '是否删除 0-否 1-是',
    `company_id`    bigint         NOT NULL DEFAULT '0' COMMENT '公司ID',
    `code`          varchar(20)    NOT NULL COMMENT '改期单编号 RS-xxx',
    `order_id`      bigint         NOT NULL DEFAULT '0' COMMENT '订单ID',
    `customer_id`   bigint         NOT NULL DEFAULT '0' COMMENT '客户ID',
    `original_date` varchar(20)             DEFAULT NULL COMMENT '原拍摄日期',
    `original_time` varchar(20)             DEFAULT NULL COMMENT '原时间段',
    `new_date`      varchar(20)             DEFAULT NULL COMMENT '新拍摄日期',
    `new_time`      varchar(20)             DEFAULT NULL COMMENT '新时间段',
    `fee_type`      tinyint        NOT NULL DEFAULT '1' COMMENT '费用类型 1-免费 2-收调度费 3-不可改期',
    `fee_amount`    decimal(12, 2) NOT NULL DEFAULT '0.00' COMMENT '调度费金额',
    `reason_label`  varchar(50)             DEFAULT NULL COMMENT '改期原因分类(时间冲突/天气原因/场地问题/与客户协商)',
    `reason`        varchar(500)            DEFAULT NULL COMMENT '改期原因说明',
    `status`        tinyint        NOT NULL DEFAULT '1' COMMENT '状态 1-待确认 2-已同意 3-已拒绝 4-已取消',
    `apply_source`  tinyint        NOT NULL DEFAULT '2' COMMENT '申请来源 1-摄影师发起 2-客户申请',
    `audit_by`      bigint         NOT NULL DEFAULT '0' COMMENT '审批人ID',
    `audit_name`    varchar(50)             DEFAULT NULL COMMENT '审批人',
    `audit_at`      datetime                DEFAULT NULL COMMENT '审批时间',
    `audit_remark`  varchar(200)            DEFAULT NULL COMMENT '审批备注',
    PRIMARY KEY (`id`),
    UNIQUE KEY `uk_reschedule_code` (`code`,`deleted`),
    KEY             `idx_reschedule_order` (`order_id`),
    KEY             `idx_reschedule_company` (`company_id`),
    KEY             `idx_reschedule_customer` (`customer_id`),
    KEY             `idx_reschedule_status` (`status`),
    KEY             `idx_reschedule_deleted` (`deleted`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8 COMMENT='订单改期单';

-- 订单评价（客户成片交付后评价摄影师）
DROP TABLE IF EXISTS `biz_order_review`;
CREATE TABLE `biz_order_review`
(
    `id`            bigint   NOT NULL AUTO_INCREMENT COMMENT '主键ID',
    `created_by`    bigint   NOT NULL DEFAULT '0' COMMENT '创建人',
    `created_at`    datetime NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `updated_by`    bigint   NOT NULL DEFAULT '0' COMMENT '修改人',
    `updated_at`    datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '修改时间',
    `deleted`       bigint   NOT NULL DEFAULT '0' COMMENT '是否删除 0-否 1-是',
    `company_id`    bigint   NOT NULL DEFAULT '0' COMMENT '公司ID',
    `order_id`      bigint   NOT NULL DEFAULT '0' COMMENT '订单ID',
    `customer_id`   bigint   NOT NULL DEFAULT '0' COMMENT '客户ID',
    `customer_name` varchar(50)       DEFAULT NULL COMMENT '客户姓名(快照)',
    `rating`        tinyint  NOT NULL DEFAULT '5' COMMENT '评分 1-5',
    `content`       varchar(500)      DEFAULT NULL COMMENT '评价内容',
    `images`        text COMMENT '评价图片(逗号分隔)',
    `is_anonymous`  tinyint  NOT NULL DEFAULT '0' COMMENT '是否匿名 0-否 1-是',
    `reply`         varchar(500)      DEFAULT NULL COMMENT '摄影师回复',
    `reply_at`      datetime          DEFAULT NULL COMMENT '回复时间',
    PRIMARY KEY (`id`),
    KEY             `idx_review_order` (`order_id`),
    KEY             `idx_review_company` (`company_id`),
    KEY             `idx_review_customer` (`customer_id`),
    KEY             `idx_review_rating` (`rating`),
    KEY             `idx_review_deleted` (`deleted`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8 COMMENT='订单评价';

-- 订单操作日志
DROP TABLE IF EXISTS `biz_order_log`;
CREATE TABLE `biz_order_log`
(
    `id`            bigint   NOT NULL AUTO_INCREMENT COMMENT '主键ID',
    `created_by`    bigint   NOT NULL DEFAULT '0' COMMENT '创建人',
    `created_at`    datetime NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `updated_by`    bigint   NOT NULL DEFAULT '0' COMMENT '修改人',
    `updated_at`    datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '修改时间',
    `deleted`       bigint   NOT NULL DEFAULT '0' COMMENT '是否删除 0-否 1-是',
    `company_id`    bigint   NOT NULL DEFAULT '0' COMMENT '公司ID',
    `order_id`      bigint   NOT NULL DEFAULT '0' COMMENT '订单ID',
    `action`        varchar(50)       DEFAULT NULL COMMENT '操作动作',
    `from_status`   tinyint           DEFAULT NULL COMMENT '原状态',
    `to_status`     tinyint           DEFAULT NULL COMMENT '新状态',
    `content`       varchar(500)      DEFAULT NULL COMMENT '操作内容',
    `operator_id`   bigint   NOT NULL DEFAULT '0' COMMENT '操作人ID',
    `operator_name` varchar(50)       DEFAULT NULL COMMENT '操作人',
    PRIMARY KEY (`id`),
    KEY             `idx_orderlog_order` (`order_id`),
    KEY             `idx_orderlog_company` (`company_id`),
    KEY             `idx_orderlog_deleted` (`deleted`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8 COMMENT='订单操作日志';

-- ---------------------------------------------------------------------
-- 五、交付（选片 / 精修 / 成片确认）
-- ---------------------------------------------------------------------

-- 交付单（选片精修交付流程）
DROP TABLE IF EXISTS `biz_delivery`;
CREATE TABLE `biz_delivery`
(
    `id`                    bigint         NOT NULL AUTO_INCREMENT COMMENT '主键ID',
    `created_by`            bigint         NOT NULL DEFAULT '0' COMMENT '创建人',
    `created_at`            datetime       NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `updated_by`            bigint         NOT NULL DEFAULT '0' COMMENT '修改人',
    `updated_at`            datetime       NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '修改时间',
    `deleted`               bigint         NOT NULL DEFAULT '0' COMMENT '是否删除 0-否 1-是',
    `company_id`            bigint         NOT NULL DEFAULT '0' COMMENT '公司ID',
    `code`                  varchar(20)    NOT NULL COMMENT '交付单编号 DV-xxx',
    `order_id`              bigint         NOT NULL DEFAULT '0' COMMENT '订单ID',
    `customer_id`           bigint         NOT NULL DEFAULT '0' COMMENT '客户ID',
    `customer_name`         varchar(50)             DEFAULT NULL COMMENT '客户姓名(快照)',
    `stage`                 tinyint        NOT NULL DEFAULT '1' COMMENT '阶段 1-待上传样片 2-客户选片中 3-精修进行中 4-待确认交付 5-已交付',
    `raw_count`             int            NOT NULL DEFAULT '0' COMMENT '原片数量(计划)',
    `retouch_target`        int            NOT NULL DEFAULT '0' COMMENT '计划精修张数',
    `sample_count`          int            NOT NULL DEFAULT '0' COMMENT '样片数量',
    `selected_count`        int            NOT NULL DEFAULT '0' COMMENT '客户已选张数',
    `select_deadline`       datetime                DEFAULT NULL COMMENT '选片截止时间(逾期默认全选)',
    `extra_selected_count`  int            NOT NULL DEFAULT '0' COMMENT '加选张数(超出套餐精修数)',
    `extra_fee`             decimal(12, 2) NOT NULL DEFAULT '0.00' COMMENT '加选费用(计入尾款)',
    `extra_confirmed`       tinyint        NOT NULL DEFAULT '0' COMMENT '客户是否确认加片 0-待确认 1-已确认',
    `retouch_version`       int            NOT NULL DEFAULT '1' COMMENT '精修轮次(最终成片 V1/V2...)',
    `sent_final_at`         datetime                DEFAULT NULL COMMENT '发送最终确认时间',
    `customer_confirmed_at` datetime                DEFAULT NULL COMMENT '客户确认成片时间',
    `retouched_count`       int            NOT NULL DEFAULT '0' COMMENT '精修完成张数',
    `selected_at`           datetime                DEFAULT NULL COMMENT '选片完成时间',
    `delivered_at`          datetime                DEFAULT NULL COMMENT '交付时间',
    `remark`                varchar(500)            DEFAULT NULL COMMENT '备注',
    `operator_id`           bigint         NOT NULL DEFAULT '0' COMMENT '当前处理人ID',
    PRIMARY KEY (`id`),
    UNIQUE KEY `uk_delivery_code` (`code`,`deleted`),
    KEY                     `idx_delivery_order` (`order_id`),
    KEY                     `idx_delivery_company` (`company_id`),
    KEY                     `idx_delivery_customer` (`customer_id`),
    KEY                     `idx_delivery_stage` (`stage`),
    KEY                     `idx_delivery_deleted` (`deleted`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8 COMMENT='交付单';

-- 交付明细（样片/精修文件）
DROP TABLE IF EXISTS `biz_delivery_item`;
CREATE TABLE `biz_delivery_item`
(
    `id`                bigint       NOT NULL AUTO_INCREMENT COMMENT '主键ID',
    `created_by`        bigint       NOT NULL DEFAULT '0' COMMENT '创建人',
    `created_at`        datetime     NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `updated_by`        bigint       NOT NULL DEFAULT '0' COMMENT '修改人',
    `updated_at`        datetime     NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '修改时间',
    `deleted`           bigint       NOT NULL DEFAULT '0' COMMENT '是否删除 0-否 1-是',
    `company_id`        bigint       NOT NULL DEFAULT '0' COMMENT '公司ID',
    `delivery_id`       bigint       NOT NULL DEFAULT '0' COMMENT '交付单ID',
    `order_id`          bigint       NOT NULL DEFAULT '0' COMMENT '订单ID',
    `url`               varchar(500) NOT NULL COMMENT '文件地址',
    `file_type`         tinyint               DEFAULT NULL COMMENT '类型 1-图片 2-视频 3-文件',
    `kind`              tinyint               DEFAULT NULL COMMENT '用途 1-样片 2-已选 3-精修成品',
    `is_selected`       tinyint      NOT NULL DEFAULT '0' COMMENT '客户是否选中 0-否 1-是',
    `feedback_content`  varchar(500)          DEFAULT NULL COMMENT '客户修图反馈内容',
    `feedback_types`    varchar(100)          DEFAULT NULL COMMENT '反馈修改类型(逗号分隔: 局部修饰/颜色调整/构图裁剪/其他)',
    `feedback_priority` varchar(10)           DEFAULT NULL COMMENT '反馈优先级 normal-一般 important-重要 urgent-紧急',
    `feedback_status`   tinyint      NOT NULL DEFAULT '0' COMMENT '反馈状态 0-无反馈 1-待处理 2-已处理',
    `handled_at`        datetime              DEFAULT NULL COMMENT '反馈处理时间',
    `handle_remark`     varchar(200)          DEFAULT NULL COMMENT '处理备注(如确认加项)',
    `filename`          varchar(200)          DEFAULT NULL COMMENT '原始文件名',
    `size`              bigint                DEFAULT NULL COMMENT '文件大小(字节)',
    PRIMARY KEY (`id`),
    KEY                 `idx_delivery_item_delivery` (`delivery_id`),
    KEY                 `idx_delivery_item_order` (`order_id`),
    KEY                 `idx_delivery_item_company` (`company_id`),
    KEY                 `idx_delivery_item_deleted` (`deleted`),
    KEY                 `idx_delivery_item_selected` (`is_selected`),
    KEY                 `idx_delivery_item_feedback` (`feedback_status`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8 COMMENT='交付明细';

-- ---------------------------------------------------------------------
-- 六、作品与档期
-- ---------------------------------------------------------------------

-- 作品集
DROP TABLE IF EXISTS `biz_asset`;
CREATE TABLE `biz_asset`
(
    `id`            bigint      NOT NULL AUTO_INCREMENT COMMENT '主键ID',
    `created_by`    bigint      NOT NULL DEFAULT '0' COMMENT '创建人',
    `created_at`    datetime    NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `updated_by`    bigint      NOT NULL DEFAULT '0' COMMENT '修改人',
    `updated_at`    datetime    NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '修改时间',
    `deleted`       bigint      NOT NULL DEFAULT '0' COMMENT '是否删除 0-否 1-是',
    `company_id`    bigint      NOT NULL DEFAULT '0' COMMENT '公司ID',
    `code`          varchar(20) NOT NULL COMMENT '作品编号 WK-xxx',
    `title`         varchar(100)         DEFAULT NULL COMMENT '作品标题',
    `category`      varchar(50)          DEFAULT NULL COMMENT '作品类型(婚纱/写真/儿童/全家福/活动跟拍等)',
    `cover`         varchar(500)         DEFAULT NULL COMMENT '封面图',
    `images`        text COMMENT '作品图片(逗号分隔)',
    `description`   varchar(1000)        DEFAULT NULL COMMENT '作品描述',
    `photographer`  varchar(50)          DEFAULT NULL COMMENT '摄影师',
    `model`         varchar(50)          DEFAULT NULL COMMENT '模特',
    `location`      varchar(100)         DEFAULT NULL COMMENT '拍摄地点',
    `status`        tinyint     NOT NULL DEFAULT '1' COMMENT '状态 1-草稿 2-已发布',
    `visibility`    tinyint     NOT NULL DEFAULT '1' COMMENT '可见性 1-公开 2-未公开',
    `featured`      tinyint     NOT NULL DEFAULT '0' COMMENT '精选展示(主页顶部) 0-否 1-是',
    `authorization` tinyint     NOT NULL DEFAULT '2' COMMENT '客户授权 1-待授权 2-已授权',
    `shoot_date`    varchar(20)          DEFAULT NULL COMMENT '拍摄日期',
    `package_ids`   varchar(200)         DEFAULT NULL COMMENT '关联套餐ID(逗号分隔)',
    `view_count`    bigint      NOT NULL DEFAULT '0' COMMENT '浏览数',
    `published_at`  datetime             DEFAULT NULL COMMENT '发布时间',
    PRIMARY KEY (`id`),
    UNIQUE KEY `uk_asset_code` (`code`,`deleted`),
    KEY             `idx_asset_company` (`company_id`),
    KEY             `idx_asset_category` (`category`),
    KEY             `idx_asset_status` (`status`),
    KEY             `idx_asset_deleted` (`deleted`),
    KEY             `idx_asset_featured` (`featured`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8 COMMENT='作品集';

-- 档期锁定
DROP TABLE IF EXISTS `biz_calendar_block`;
CREATE TABLE `biz_calendar_block`
(
    `id`              bigint   NOT NULL AUTO_INCREMENT COMMENT '主键ID',
    `created_by`      bigint   NOT NULL DEFAULT '0' COMMENT '创建人',
    `created_at`      datetime NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `updated_by`      bigint   NOT NULL DEFAULT '0' COMMENT '修改人',
    `updated_at`      datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '修改时间',
    `deleted`         bigint   NOT NULL DEFAULT '0' COMMENT '是否删除 0-否 1-是',
    `company_id`      bigint   NOT NULL DEFAULT '0' COMMENT '公司ID',
    `store_id`        bigint   NOT NULL DEFAULT '0' COMMENT '所属门店ID',
    `order_id`        bigint   NOT NULL DEFAULT '0' COMMENT '关联订单ID',
    `customer_id`     bigint   NOT NULL DEFAULT '0' COMMENT '关联客户ID',
    `customer_name`   varchar(50)       DEFAULT NULL COMMENT '客户姓名(快照)',
    `date`            varchar(10)       DEFAULT NULL COMMENT '拍摄日期 yyyy-MM-dd',
    `time_range`      varchar(50)       DEFAULT NULL COMMENT '时间段',
    `project_type`    varchar(50)       DEFAULT NULL COMMENT '项目类型',
    `photographer_id` bigint   NOT NULL DEFAULT '0' COMMENT '摄影师ID',
    `photographer`    varchar(50)       DEFAULT NULL COMMENT '摄影师姓名',
    `status`          tinyint  NOT NULL DEFAULT '1' COMMENT '状态 1-已锁定 2-已取消',
    `remark`          varchar(200)      DEFAULT NULL COMMENT '备注',
    PRIMARY KEY (`id`),
    KEY               `idx_block_company` (`company_id`),
    KEY               `idx_block_store` (`store_id`),
    KEY               `idx_block_order` (`order_id`),
    KEY               `idx_block_date` (`date`),
    KEY               `idx_block_photographer` (`photographer_id`),
    KEY               `idx_block_deleted` (`deleted`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8 COMMENT='档期锁定';

-- 档期时段模板（摄影师每周可约时段规则，客户预约页据此生成可约日历）
DROP TABLE IF EXISTS `biz_slot_template`;
CREATE TABLE `biz_slot_template`
(
    `id`              bigint      NOT NULL AUTO_INCREMENT COMMENT '主键ID',
    `created_by`      bigint      NOT NULL DEFAULT '0' COMMENT '创建人',
    `created_at`      datetime    NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `updated_by`      bigint      NOT NULL DEFAULT '0' COMMENT '修改人',
    `updated_at`      datetime    NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '修改时间',
    `deleted`         bigint      NOT NULL DEFAULT '0' COMMENT '是否删除 0-否 1-是',
    `company_id`      bigint      NOT NULL DEFAULT '0' COMMENT '公司ID',
    `store_id`        bigint      NOT NULL DEFAULT '0' COMMENT '所属门店ID',
    `photographer_id` bigint      NOT NULL DEFAULT '0' COMMENT '摄影师ID(0=全店通用)',
    `weekday`         tinyint     NOT NULL DEFAULT '1' COMMENT '星期几 0-周日 1-周一 ... 6-周六',
    `start_time`      varchar(10) NOT NULL COMMENT '开始时间 HH:mm',
    `end_time`        varchar(10) NOT NULL COMMENT '结束时间 HH:mm',
    `status`          tinyint     NOT NULL DEFAULT '1' COMMENT '状态 1-启用 0-停用',
    PRIMARY KEY (`id`),
    KEY               `idx_slot_tmpl_company` (`company_id`),
    KEY               `idx_slot_tmpl_photographer` (`photographer_id`),
    KEY               `idx_slot_tmpl_weekday` (`weekday`),
    KEY               `idx_slot_tmpl_deleted` (`deleted`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8 COMMENT='档期时段模板';

-- ---------------------------------------------------------------------
-- 七、基础配置与附件
-- ---------------------------------------------------------------------

-- 收款方式
DROP TABLE IF EXISTS `biz_payment_method`;
CREATE TABLE `biz_payment_method`
(
    `id`           bigint      NOT NULL AUTO_INCREMENT COMMENT '主键ID',
    `created_by`   bigint      NOT NULL DEFAULT '0' COMMENT '创建人',
    `created_at`   datetime    NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `updated_by`   bigint      NOT NULL DEFAULT '0' COMMENT '修改人',
    `updated_at`   datetime    NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '修改时间',
    `deleted`      bigint      NOT NULL DEFAULT '0' COMMENT '是否删除 0-否 1-是',
    `company_id`   bigint      NOT NULL DEFAULT '0' COMMENT '公司ID',
    `name`         varchar(50) NOT NULL COMMENT '收款方式名称',
    `type`         varchar(20)          DEFAULT NULL COMMENT '类型 wechat-微信 alipay-支付宝 bank-银行转账 cash-现金 other-其他',
    `account_name` varchar(100)         DEFAULT NULL COMMENT '收款账户名称',
    `account_no`   varchar(100)         DEFAULT NULL COMMENT '收款账号',
    `qrcode`       varchar(500)         DEFAULT NULL COMMENT '收款二维码',
    `status`       tinyint     NOT NULL DEFAULT '1' COMMENT '状态 1-启用 0-停用',
    `sort`         int         NOT NULL DEFAULT '0' COMMENT '排序',
    PRIMARY KEY (`id`),
    KEY            `idx_paymethod_company` (`company_id`),
    KEY            `idx_paymethod_deleted` (`deleted`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8 COMMENT='收款方式';

-- 工作室设置（预约主页/接单规则/改期政策，company 级唯一）
DROP TABLE IF EXISTS `biz_studio_setting`;
CREATE TABLE `biz_studio_setting`
(
    `id`                    bigint        NOT NULL AUTO_INCREMENT COMMENT '主键ID',
    `created_by`            bigint        NOT NULL DEFAULT '0' COMMENT '创建人',
    `created_at`            datetime      NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `updated_by`            bigint        NOT NULL DEFAULT '0' COMMENT '修改人',
    `updated_at`            datetime      NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '修改时间',
    `deleted`               bigint        NOT NULL DEFAULT '0' COMMENT '是否删除 0-否 1-是',
    `company_id`            bigint        NOT NULL DEFAULT '0' COMMENT '公司ID',
    `slogan`                varchar(200)           DEFAULT NULL COMMENT '宣传语',
    `intro`                 varchar(1000)          DEFAULT NULL COMMENT '工作室简介',
    `homepage_slug`         varchar(50)            DEFAULT NULL COMMENT '预约主页短链标识',
    `accept_new`            tinyint       NOT NULL DEFAULT '1' COMMENT '接收新预约 0-暂停 1-接收',
    `lock_minutes`          int           NOT NULL DEFAULT '15' COMMENT '下单临时锁定时长(分钟)',
    `reschedule_free_hours` int           NOT NULL DEFAULT '72' COMMENT '免费改期剩余小时阈值',
    `reschedule_fee_rate`   decimal(5, 2) NOT NULL DEFAULT '20.00' COMMENT '改期调度费率(%)',
    `reschedule_min_hours`  int           NOT NULL DEFAULT '24' COMMENT '距拍摄不足该小时数不可改期',
    `select_deadline_hours` int           NOT NULL DEFAULT '72' COMMENT '上传样片后选片截止(小时)',
    `retain_days`           int           NOT NULL DEFAULT '30' COMMENT '未选原片保留天数',
    `faq`                   text COMMENT '常见问题(JSON数组: [{q,a}])',
    `service_flow`          text COMMENT '服务流程(JSON数组)',
    PRIMARY KEY (`id`),
    UNIQUE KEY `uk_studio_setting_company` (`company_id`,`deleted`),
    KEY                     `idx_studio_setting_deleted` (`deleted`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8 COMMENT='工作室设置(预约主页/接单规则)';

-- 上传文件记录
DROP TABLE IF EXISTS `biz_upload`;
CREATE TABLE `biz_upload`
(
    `id`         bigint       NOT NULL AUTO_INCREMENT COMMENT '主键ID',
    `created_by` bigint       NOT NULL DEFAULT '0' COMMENT '创建人',
    `created_at` datetime     NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `updated_by` bigint       NOT NULL DEFAULT '0' COMMENT '修改人',
    `updated_at` datetime     NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '修改时间',
    `deleted`    bigint       NOT NULL DEFAULT '0' COMMENT '是否删除 0-否 1-是',
    `company_id` bigint       NOT NULL DEFAULT '0' COMMENT '公司ID',
    `store_id`   bigint       NOT NULL DEFAULT '0' COMMENT '所属门店ID',
    `biz_type`   varchar(50)           DEFAULT NULL COMMENT '业务类型(订单/交付/作品等)',
    `biz_id`     bigint       NOT NULL DEFAULT '0' COMMENT '业务ID',
    `file_type`  tinyint               DEFAULT NULL COMMENT '文件类型 1-图片 2-视频 3-文件',
    `file_name`  varchar(200)          DEFAULT NULL COMMENT '原始文件名',
    `file_url`   varchar(500) NOT NULL COMMENT '文件访问地址',
    `file_path`  varchar(500)          DEFAULT NULL COMMENT '服务器存储路径',
    `size`       bigint                DEFAULT NULL COMMENT '文件大小(字节)',
    `upload_by`  bigint       NOT NULL DEFAULT '0' COMMENT '上传人ID',
    PRIMARY KEY (`id`),
    KEY          `idx_upload_company` (`company_id`),
    KEY          `idx_upload_store` (`store_id`),
    KEY          `idx_upload_biz` (`biz_type`,`biz_id`),
    KEY          `idx_upload_deleted` (`deleted`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8 COMMENT='上传文件记录';
