-- =====================================================================
-- SLOT 摄影工作室管理系统 增量升级脚本（客户端三端能力：小程序/APP/H5）
-- 依据：原型《SLOT_PC后台管理》《摄影小工具H5》《摄影小工具移动端》
-- 日期：2026-09-07
-- 约定：所有新增表/字段沿用 DDL 通用约定（created_by/created_at/updated_by/updated_at/deleted/company_id）
-- 执行方式：在已有库上按顺序执行，幂等性不做保证（字段存在会报错，属预期）
-- =====================================================================

USE `photography`;
SET NAMES utf8mb4;

-- ---------------------------------------------------------------------
-- 一、既有表字段扩展（ALTER）
-- ---------------------------------------------------------------------

-- 订单：客户预约入口 + 拍前准备 + 选片截止 + 下载有效期 + 归档
ALTER TABLE `biz_order`
  MODIFY COLUMN `status` TINYINT NOT NULL DEFAULT 1 COMMENT '订单状态 0-待确认(客户预约) 1-待定金 2-待拍摄 3-拍摄中 4-精修中 5-待交付 6-已完成 7-已取消',
  ADD COLUMN `people_count` VARCHAR(20) DEFAULT NULL COMMENT '拍摄人数(如 2大1小)' AFTER `shoot_address`,
  ADD COLUMN `shoot_style` VARCHAR(50) DEFAULT NULL COMMENT '拍摄风格' AFTER `people_count`,
  ADD COLUMN `source_type` TINYINT NOT NULL DEFAULT 1 COMMENT '订单来源 1-管理端录入 2-客户预约(H5/小程序) 3-线索报价转化' AFTER `remark`,
  ADD COLUMN `prep_content` TEXT COMMENT '拍前准备清单内容(时间/地点/服装/交通/注意事项)' AFTER `source_type`,
  ADD COLUMN `prep_read_at` DATETIME DEFAULT NULL COMMENT '客户确认阅读拍前准备时间' AFTER `prep_content`,
  ADD COLUMN `select_deadline` DATETIME DEFAULT NULL COMMENT '选片截止时间(逾期默认全选)' AFTER `prep_read_at`,
  ADD COLUMN `delivery_expire_at` DATETIME DEFAULT NULL COMMENT '成片下载有效期至' AFTER `select_deadline`,
  ADD COLUMN `is_archived` TINYINT NOT NULL DEFAULT 0 COMMENT '是否已归档 0-否 1-是' AFTER `delivery_expire_at`,
  ADD COLUMN `archived_at` DATETIME DEFAULT NULL COMMENT '归档时间' AFTER `is_archived`;

-- 报价单：有效期/时长/地点/加项/转化（客户 H5 查看/确认报价）
ALTER TABLE `biz_quote`
  ADD COLUMN `valid_until` DATETIME DEFAULT NULL COMMENT '报价有效期至' AFTER `status`,
  ADD COLUMN `shoot_time` VARCHAR(20) DEFAULT NULL COMMENT '拍摄时间段' AFTER `shoot_date`,
  ADD COLUMN `duration_hours` DECIMAL(5,2) DEFAULT NULL COMMENT '拍摄时长(小时)' AFTER `shoot_time`,
  ADD COLUMN `location` VARCHAR(200) DEFAULT NULL COMMENT '拍摄地点' AFTER `duration_hours`,
  ADD COLUMN `location_type` VARCHAR(20) DEFAULT NULL COMMENT '地点类型 outdoor-外拍 studio-合作影棚' AFTER `location`,
  ADD COLUMN `location_fee` DECIMAL(12,2) NOT NULL DEFAULT 0.00 COMMENT '场地费' AFTER `location_type`,
  ADD COLUMN `people_count` VARCHAR(20) DEFAULT NULL COMMENT '拍摄人数' AFTER `location_fee`,
  ADD COLUMN `addons` TEXT COMMENT '加项清单(JSON数组: [{name,price,qty}])' AFTER `people_count`,
  ADD COLUMN `accept_at` DATETIME DEFAULT NULL COMMENT '客户接受时间' AFTER `addons`,
  ADD COLUMN `order_id` BIGINT NOT NULL DEFAULT 0 COMMENT '转化生成的订单ID' AFTER `accept_at`,
  ADD KEY `idx_quote_order` (`order_id`);

-- 交付单：选片截止/加选/精修轮次/最终确认（客户选片与确认成片）
ALTER TABLE `biz_delivery`
  ADD COLUMN `select_deadline` DATETIME DEFAULT NULL COMMENT '选片截止时间(逾期默认全选)' AFTER `selected_count`,
  ADD COLUMN `extra_selected_count` INT NOT NULL DEFAULT 0 COMMENT '加选张数(超出套餐精修数)' AFTER `select_deadline`,
  ADD COLUMN `extra_fee` DECIMAL(12,2) NOT NULL DEFAULT 0.00 COMMENT '加选费用(计入尾款)' AFTER `extra_selected_count`,
  ADD COLUMN `extra_confirmed` TINYINT NOT NULL DEFAULT 0 COMMENT '客户是否确认加片 0-待确认 1-已确认' AFTER `extra_fee`,
  ADD COLUMN `retouch_version` INT NOT NULL DEFAULT 1 COMMENT '精修轮次(最终成片 V1/V2...)' AFTER `extra_confirmed`,
  ADD COLUMN `sent_final_at` DATETIME DEFAULT NULL COMMENT '发送最终确认时间' AFTER `retouch_version`,
  ADD COLUMN `customer_confirmed_at` DATETIME DEFAULT NULL COMMENT '客户确认成片时间' AFTER `sent_final_at`;

-- 交付明细：客户选片标记与修图反馈
ALTER TABLE `biz_delivery_item`
  ADD COLUMN `is_selected` TINYINT NOT NULL DEFAULT 0 COMMENT '客户是否选中 0-否 1-是' AFTER `kind`,
  ADD COLUMN `feedback_content` VARCHAR(500) DEFAULT NULL COMMENT '客户修图反馈内容' AFTER `is_selected`,
  ADD COLUMN `feedback_types` VARCHAR(100) DEFAULT NULL COMMENT '反馈修改类型(逗号分隔: 局部修饰/颜色调整/构图裁剪/其他)' AFTER `feedback_content`,
  ADD COLUMN `feedback_priority` VARCHAR(10) DEFAULT NULL COMMENT '反馈优先级 normal-一般 important-重要 urgent-紧急' AFTER `feedback_types`,
  ADD COLUMN `feedback_status` TINYINT NOT NULL DEFAULT 0 COMMENT '反馈状态 0-无反馈 1-待处理 2-已处理' AFTER `feedback_priority`,
  ADD COLUMN `handled_at` DATETIME DEFAULT NULL COMMENT '反馈处理时间' AFTER `feedback_status`,
  ADD COLUMN `handle_remark` VARCHAR(200) DEFAULT NULL COMMENT '处理备注(如确认加项)' AFTER `handled_at`,
  ADD KEY `idx_delivery_item_selected` (`is_selected`),
  ADD KEY `idx_delivery_item_feedback` (`feedback_status`);

-- 套餐：适合人群/简介/原片/修改次数/交付与下载/地点模式（客户套餐详情页）
ALTER TABLE `biz_package`
  ADD COLUMN `suitable_for` VARCHAR(100) DEFAULT NULL COMMENT '适合人群(逗号分隔: 家庭/亲子/纪念日/孕妇)' AFTER `category`,
  ADD COLUMN `introduction` VARCHAR(500) DEFAULT NULL COMMENT '套餐简介(一句话)' AFTER `suitable_for`,
  ADD COLUMN `raw_count` INT NOT NULL DEFAULT 0 COMMENT '原片数量(如 100+)' AFTER `photos_included`,
  ADD COLUMN `revision_count` INT NOT NULL DEFAULT 2 COMMENT '修改次数' AFTER `raw_count`,
  ADD COLUMN `delivery_days` INT NOT NULL DEFAULT 7 COMMENT '交付周期(工作日)' AFTER `revision_count`,
  ADD COLUMN `download_days` INT NOT NULL DEFAULT 30 COMMENT '成片下载有效期(天)' AFTER `delivery_days`,
  ADD COLUMN `location_mode` VARCHAR(20) NOT NULL DEFAULT 'custom' COMMENT '地点模式 fixed-固定地点 custom-客户可选' AFTER `download_days`,
  ADD COLUMN `locations` TEXT COMMENT '可选地点(JSON数组: [{name,extra_fee,is_default}])' AFTER `location_mode`,
  ADD COLUMN `reschedule_policy` VARCHAR(200) DEFAULT NULL COMMENT '改期政策文案' AFTER `locations`,
  ADD COLUMN `cancel_policy` VARCHAR(200) DEFAULT NULL COMMENT '取消政策文案' AFTER `reschedule_policy`;

-- 作品：可见性/精选/授权（摄影师作品集 + 客户作品列表）
ALTER TABLE `biz_asset`
  ADD COLUMN `visibility` TINYINT NOT NULL DEFAULT 1 COMMENT '可见性 1-公开 2-未公开' AFTER `status`,
  ADD COLUMN `featured` TINYINT NOT NULL DEFAULT 0 COMMENT '精选展示(主页顶部) 0-否 1-是' AFTER `visibility`,
  ADD COLUMN `authorization` TINYINT NOT NULL DEFAULT 2 COMMENT '客户授权 1-待授权 2-已授权' AFTER `featured`,
  ADD COLUMN `shoot_date` VARCHAR(20) DEFAULT NULL COMMENT '拍摄日期' AFTER `authorization`,
  ADD COLUMN `package_ids` VARCHAR(200) DEFAULT NULL COMMENT '关联套餐ID(逗号分隔)' AFTER `shoot_date`,
  ADD KEY `idx_asset_featured` (`featured`);

-- 客户：小程序/客户端登录身份
ALTER TABLE `crm_customer`
  ADD COLUMN `openid` VARCHAR(64) DEFAULT NULL COMMENT '微信小程序openid' AFTER `avatar`,
  ADD COLUMN `unionid` VARCHAR(64) DEFAULT NULL COMMENT '微信unionid' AFTER `openid`,
  ADD COLUMN `is_verified` TINYINT NOT NULL DEFAULT 0 COMMENT '手机号是否已验证 0-否 1-是' AFTER `unionid`,
  ADD KEY `idx_customer_openid` (`openid`);

-- 退款：三步退款流（凭证上传/客户确认/申请来源）
ALTER TABLE `biz_order_refund`
  ADD COLUMN `reason_label` VARCHAR(50) DEFAULT NULL COMMENT '取消原因分类(时间冲突/预算原因/找到其他摄影师/其他)' AFTER `reason`,
  ADD COLUMN `apply_source` TINYINT NOT NULL DEFAULT 1 COMMENT '申请来源 1-管理端 2-客户H5/小程序' AFTER `reason_label`,
  ADD COLUMN `voucher` VARCHAR(500) DEFAULT NULL COMMENT '退款转账凭证截图' AFTER `apply_source`,
  ADD COLUMN `customer_confirm_at` DATETIME DEFAULT NULL COMMENT '客户确认收到退款时间' AFTER `voucher`;

-- 站内通知：支持客户接收
ALTER TABLE `sys_notification`
  ADD COLUMN `receiver_type` TINYINT NOT NULL DEFAULT 1 COMMENT '接收人类型 1-员工 2-客户' AFTER `receiver_id`,
  ADD KEY `idx_notice_receiver_type` (`receiver_type`, `receiver_id`);

-- ---------------------------------------------------------------------
-- 二、新增表
-- ---------------------------------------------------------------------

-- 改期单（客户申请/摄影师发起 → 摄影师审批 → 档期变更）
DROP TABLE IF EXISTS `biz_order_reschedule`;
CREATE TABLE `biz_order_reschedule` (
  `id`              BIGINT       NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `created_by`      BIGINT       NOT NULL DEFAULT 0 COMMENT '创建人',
  `created_at`      DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `updated_by`      BIGINT       NOT NULL DEFAULT 0 COMMENT '修改人',
  `updated_at`      DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '修改时间',
  `deleted`         BIGINT       NOT NULL DEFAULT 0 COMMENT '是否删除 0-否 1-是',
  `company_id`      BIGINT       NOT NULL DEFAULT 0 COMMENT '公司ID',
  `code`            VARCHAR(20)  NOT NULL COMMENT '改期单编号 RS-xxx',
  `order_id`        BIGINT       NOT NULL DEFAULT 0 COMMENT '订单ID',
  `customer_id`     BIGINT       NOT NULL DEFAULT 0 COMMENT '客户ID',
  `original_date`   VARCHAR(20)  DEFAULT NULL COMMENT '原拍摄日期',
  `original_time`   VARCHAR(20)  DEFAULT NULL COMMENT '原时间段',
  `new_date`        VARCHAR(20)  DEFAULT NULL COMMENT '新拍摄日期',
  `new_time`        VARCHAR(20)  DEFAULT NULL COMMENT '新时间段',
  `fee_type`        TINYINT      NOT NULL DEFAULT 1 COMMENT '费用类型 1-免费 2-收调度费 3-不可改期',
  `fee_amount`      DECIMAL(12,2) NOT NULL DEFAULT 0.00 COMMENT '调度费金额',
  `reason_label`    VARCHAR(50)  DEFAULT NULL COMMENT '改期原因分类(时间冲突/天气原因/场地问题/与客户协商)',
  `reason`          VARCHAR(500) DEFAULT NULL COMMENT '改期原因说明',
  `status`          TINYINT      NOT NULL DEFAULT 1 COMMENT '状态 1-待确认 2-已同意 3-已拒绝 4-已取消',
  `apply_source`    TINYINT      NOT NULL DEFAULT 2 COMMENT '申请来源 1-摄影师发起 2-客户申请',
  `audit_by`        BIGINT       NOT NULL DEFAULT 0 COMMENT '审批人ID',
  `audit_name`      VARCHAR(50)  DEFAULT NULL COMMENT '审批人',
  `audit_at`        DATETIME     DEFAULT NULL COMMENT '审批时间',
  `audit_remark`    VARCHAR(200) DEFAULT NULL COMMENT '审批备注',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_reschedule_code` (`code`, `deleted`),
  KEY `idx_reschedule_order` (`order_id`),
  KEY `idx_reschedule_company` (`company_id`),
  KEY `idx_reschedule_customer` (`customer_id`),
  KEY `idx_reschedule_status` (`status`),
  KEY `idx_reschedule_deleted` (`deleted`)
) ENGINE=InnoDB AUTO_INCREMENT=1 DEFAULT CHARSET=utf8 COLLATE=utf8_general_ci COMMENT='订单改期单';

-- 订单评价（客户成片交付后评价摄影师）
DROP TABLE IF EXISTS `biz_order_review`;
CREATE TABLE `biz_order_review` (
  `id`              BIGINT       NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `created_by`      BIGINT       NOT NULL DEFAULT 0 COMMENT '创建人',
  `created_at`      DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `updated_by`      BIGINT       NOT NULL DEFAULT 0 COMMENT '修改人',
  `updated_at`      DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '修改时间',
  `deleted`         BIGINT       NOT NULL DEFAULT 0 COMMENT '是否删除 0-否 1-是',
  `company_id`      BIGINT       NOT NULL DEFAULT 0 COMMENT '公司ID',
  `order_id`        BIGINT       NOT NULL DEFAULT 0 COMMENT '订单ID',
  `customer_id`     BIGINT       NOT NULL DEFAULT 0 COMMENT '客户ID',
  `customer_name`   VARCHAR(50)  DEFAULT NULL COMMENT '客户姓名(快照)',
  `rating`          TINYINT      NOT NULL DEFAULT 5 COMMENT '评分 1-5',
  `content`         VARCHAR(500) DEFAULT NULL COMMENT '评价内容',
  `images`          TEXT         COMMENT '评价图片(逗号分隔)',
  `is_anonymous`    TINYINT      NOT NULL DEFAULT 0 COMMENT '是否匿名 0-否 1-是',
  `reply`           VARCHAR(500) DEFAULT NULL COMMENT '摄影师回复',
  `reply_at`        DATETIME     DEFAULT NULL COMMENT '回复时间',
  PRIMARY KEY (`id`),
  KEY `idx_review_order` (`order_id`),
  KEY `idx_review_company` (`company_id`),
  KEY `idx_review_customer` (`customer_id`),
  KEY `idx_review_rating` (`rating`),
  KEY `idx_review_deleted` (`deleted`)
) ENGINE=InnoDB AUTO_INCREMENT=1 DEFAULT CHARSET=utf8 COLLATE=utf8_general_ci COMMENT='订单评价';

-- 定制需求（客户非套餐需求提交 → 可转化线索）
DROP TABLE IF EXISTS `biz_custom_request`;
CREATE TABLE `biz_custom_request` (
  `id`              BIGINT       NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `created_by`      BIGINT       NOT NULL DEFAULT 0 COMMENT '创建人',
  `created_at`      DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `updated_by`      BIGINT       NOT NULL DEFAULT 0 COMMENT '修改人',
  `updated_at`      DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '修改时间',
  `deleted`         BIGINT       NOT NULL DEFAULT 0 COMMENT '是否删除 0-否 1-是',
  `company_id`      BIGINT       NOT NULL DEFAULT 0 COMMENT '公司ID',
  `customer_id`     BIGINT       NOT NULL DEFAULT 0 COMMENT '客户ID(登录提交时有值)',
  `name`            VARCHAR(50)  DEFAULT NULL COMMENT '称呼',
  `mobile`          VARCHAR(20)  DEFAULT NULL COMMENT '联系电话',
  `project_type`    VARCHAR(50)  DEFAULT NULL COMMENT '拍摄类型(家庭纪念/个人写真/情侣/婚纱/儿童写真/活动跟拍/其他)',
  `expected_date`   VARCHAR(50)  DEFAULT NULL COMMENT '期望拍摄日期(如 2026年8月下旬)',
  `location`        VARCHAR(200) DEFAULT NULL COMMENT '期望拍摄地点',
  `budget_min`      DECIMAL(12,2) DEFAULT NULL COMMENT '预算下限',
  `budget_max`      DECIMAL(12,2) DEFAULT NULL COMMENT '预算上限',
  `detail`          VARCHAR(1000) DEFAULT NULL COMMENT '详细需求',
  `images`          TEXT         COMMENT '参考图片(逗号分隔)',
  `status`          TINYINT      NOT NULL DEFAULT 1 COMMENT '状态 1-待处理 2-已响应 3-已关闭',
  `lead_id`         BIGINT       NOT NULL DEFAULT 0 COMMENT '转化线索ID',
  `response`        VARCHAR(500) DEFAULT NULL COMMENT '响应说明',
  `response_by`     BIGINT       NOT NULL DEFAULT 0 COMMENT '响应人ID',
  `response_at`     DATETIME     DEFAULT NULL COMMENT '响应时间',
  PRIMARY KEY (`id`),
  KEY `idx_custom_req_company` (`company_id`),
  KEY `idx_custom_req_customer` (`customer_id`),
  KEY `idx_custom_req_status` (`status`),
  KEY `idx_custom_req_deleted` (`deleted`)
) ENGINE=InnoDB AUTO_INCREMENT=1 DEFAULT CHARSET=utf8 COLLATE=utf8_general_ci COMMENT='定制需求';

-- 订单加项（妆造/加急/收选等自定义加项；加选精修走 delivery.extra）
DROP TABLE IF EXISTS `biz_order_addon`;
CREATE TABLE `biz_order_addon` (
  `id`              BIGINT       NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `created_by`      BIGINT       NOT NULL DEFAULT 0 COMMENT '创建人',
  `created_at`      DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `updated_by`      BIGINT       NOT NULL DEFAULT 0 COMMENT '修改人',
  `updated_at`      DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '修改时间',
  `deleted`         BIGINT       NOT NULL DEFAULT 0 COMMENT '是否删除 0-否 1-是',
  `company_id`      BIGINT       NOT NULL DEFAULT 0 COMMENT '公司ID',
  `order_id`        BIGINT       NOT NULL DEFAULT 0 COMMENT '订单ID',
  `name`            VARCHAR(100) NOT NULL COMMENT '加项名称(妆造服务/加急交付/收选服务等)',
  `category`        VARCHAR(20)  DEFAULT NULL COMMENT '分类 makeup-妆造 urgency-时效 service-服务 retouch-精修',
  `price`           DECIMAL(12,2) NOT NULL DEFAULT 0.00 COMMENT '单价',
  `qty`             INT          NOT NULL DEFAULT 1 COMMENT '数量',
  `amount`          DECIMAL(12,2) NOT NULL DEFAULT 0.00 COMMENT '小计(price×qty)',
  `confirmed`       TINYINT      NOT NULL DEFAULT 0 COMMENT '客户是否确认 0-待确认 1-已确认' ,
  `remark`          VARCHAR(200) DEFAULT NULL COMMENT '备注',
  PRIMARY KEY (`id`),
  KEY `idx_addon_order` (`order_id`),
  KEY `idx_addon_company` (`company_id`),
  KEY `idx_addon_deleted` (`deleted`)
) ENGINE=InnoDB AUTO_INCREMENT=1 DEFAULT CHARSET=utf8 COLLATE=utf8_general_ci COMMENT='订单加项';

-- 线索沟通记录（客户 H5 咨询/摄影师追问/发送作品等往来消息）
DROP TABLE IF EXISTS `biz_lead_message`;
CREATE TABLE `biz_lead_message` (
  `id`              BIGINT       NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `created_by`      BIGINT       NOT NULL DEFAULT 0 COMMENT '创建人',
  `created_at`      DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `updated_by`      BIGINT       NOT NULL DEFAULT 0 COMMENT '修改人',
  `updated_at`      DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '修改时间',
  `deleted`         BIGINT       NOT NULL DEFAULT 0 COMMENT '是否删除 0-否 1-是',
  `company_id`      BIGINT       NOT NULL DEFAULT 0 COMMENT '公司ID',
  `lead_id`         BIGINT       NOT NULL DEFAULT 0 COMMENT '线索ID',
  `customer_id`     BIGINT       NOT NULL DEFAULT 0 COMMENT '客户ID',
  `direction`       TINYINT      NOT NULL DEFAULT 1 COMMENT '方向 1-客户发来 2-工作室发出',
  `channel`         VARCHAR(20)  DEFAULT NULL COMMENT '渠道 h5-预约页 wechat-微信 sms-短信 phone-电话',
  `content`         VARCHAR(1000) DEFAULT NULL COMMENT '消息内容',
  `msg_type`        TINYINT      NOT NULL DEFAULT 1 COMMENT '类型 1-文本 2-追问 3-报价通知 4-作品分享',
  `biz_id`          BIGINT       NOT NULL DEFAULT 0 COMMENT '关联业务ID(报价单ID/作品ID等)',
  PRIMARY KEY (`id`),
  KEY `idx_leadmsg_lead` (`lead_id`),
  KEY `idx_leadmsg_company` (`company_id`),
  KEY `idx_leadmsg_deleted` (`deleted`)
) ENGINE=InnoDB AUTO_INCREMENT=1 DEFAULT CHARSET=utf8 COLLATE=utf8_general_ci COMMENT='线索沟通记录';

-- AI 简报项（从客户沟通中提取/待追问的需求项）
DROP TABLE IF EXISTS `biz_lead_brief_item`;
CREATE TABLE `biz_lead_brief_item` (
  `id`              BIGINT       NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `created_by`      BIGINT       NOT NULL DEFAULT 0 COMMENT '创建人',
  `created_at`      DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `updated_by`      BIGINT       NOT NULL DEFAULT 0 COMMENT '修改人',
  `updated_at`      DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '修改时间',
  `deleted`         BIGINT       NOT NULL DEFAULT 0 COMMENT '是否删除 0-否 1-是',
  `company_id`      BIGINT       NOT NULL DEFAULT 0 COMMENT '公司ID',
  `lead_id`         BIGINT       NOT NULL DEFAULT 0 COMMENT '线索ID',
  `title`           VARCHAR(100) DEFAULT NULL COMMENT '信息项标题(具体日期/儿童作息/妆容需求等)',
  `value`           VARCHAR(500) DEFAULT NULL COMMENT '已确认的值(已确认项)',
  `question`        VARCHAR(500) DEFAULT NULL COMMENT '待追问问题',
  `ai_suggestion`   VARCHAR(500) DEFAULT NULL COMMENT 'AI 建议追问话术',
  `status`          TINYINT      NOT NULL DEFAULT 1 COMMENT '状态 1-待追问 2-已发送 3-已确认',
  `affects_pricing` TINYINT      NOT NULL DEFAULT 0 COMMENT '是否影响报价 0-否 1-是',
  `sort`            INT          NOT NULL DEFAULT 0 COMMENT '排序(按影响报价优先)',
  `sent_at`         DATETIME     DEFAULT NULL COMMENT '追问发送时间',
  `confirmed_at`    DATETIME     DEFAULT NULL COMMENT '确认时间',
  PRIMARY KEY (`id`),
  KEY `idx_briefitem_lead` (`lead_id`),
  KEY `idx_briefitem_company` (`company_id`),
  KEY `idx_briefitem_status` (`status`),
  KEY `idx_briefitem_deleted` (`deleted`)
) ENGINE=InnoDB AUTO_INCREMENT=1 DEFAULT CHARSET=utf8 COLLATE=utf8_general_ci COMMENT='线索AI简报项';

-- 档期时段模板（摄影师每周可约时段规则，客户预约页据此生成可约日历）
DROP TABLE IF EXISTS `biz_slot_template`;
CREATE TABLE `biz_slot_template` (
  `id`              BIGINT       NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `created_by`      BIGINT       NOT NULL DEFAULT 0 COMMENT '创建人',
  `created_at`      DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `updated_by`      BIGINT       NOT NULL DEFAULT 0 COMMENT '修改人',
  `updated_at`      DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '修改时间',
  `deleted`         BIGINT       NOT NULL DEFAULT 0 COMMENT '是否删除 0-否 1-是',
  `company_id`      BIGINT       NOT NULL DEFAULT 0 COMMENT '公司ID',
  `store_id`        BIGINT       NOT NULL DEFAULT 0 COMMENT '所属门店ID',
  `photographer_id` BIGINT       NOT NULL DEFAULT 0 COMMENT '摄影师ID(0=全店通用)',
  `weekday`         TINYINT      NOT NULL DEFAULT 1 COMMENT '星期几 0-周日 1-周一 ... 6-周六',
  `start_time`      VARCHAR(10)  NOT NULL COMMENT '开始时间 HH:mm',
  `end_time`        VARCHAR(10)  NOT NULL COMMENT '结束时间 HH:mm',
  `status`          TINYINT      NOT NULL DEFAULT 1 COMMENT '状态 1-启用 0-停用',
  PRIMARY KEY (`id`),
  KEY `idx_slot_tmpl_company` (`company_id`),
  KEY `idx_slot_tmpl_photographer` (`photographer_id`),
  KEY `idx_slot_tmpl_weekday` (`weekday`),
  KEY `idx_slot_tmpl_deleted` (`deleted`)
) ENGINE=InnoDB AUTO_INCREMENT=1 DEFAULT CHARSET=utf8 COLLATE=utf8_general_ci COMMENT='档期时段模板';

-- 工作室设置（预约主页/接单规则/改期政策，company 级唯一）
DROP TABLE IF EXISTS `biz_studio_setting`;
CREATE TABLE `biz_studio_setting` (
  `id`                  BIGINT       NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `created_by`          BIGINT       NOT NULL DEFAULT 0 COMMENT '创建人',
  `created_at`          DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `updated_by`          BIGINT       NOT NULL DEFAULT 0 COMMENT '修改人',
  `updated_at`          DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '修改时间',
  `deleted`             BIGINT       NOT NULL DEFAULT 0 COMMENT '是否删除 0-否 1-是',
  `company_id`          BIGINT       NOT NULL DEFAULT 0 COMMENT '公司ID',
  `slogan`              VARCHAR(200) DEFAULT NULL COMMENT '宣传语',
  `intro`               VARCHAR(1000) DEFAULT NULL COMMENT '工作室简介',
  `homepage_slug`       VARCHAR(50)  DEFAULT NULL COMMENT '预约主页短链标识',
  `accept_new`          TINYINT      NOT NULL DEFAULT 1 COMMENT '接收新预约 0-暂停 1-接收',
  `lock_minutes`        INT          NOT NULL DEFAULT 15 COMMENT '下单临时锁定时长(分钟)',
  `reschedule_free_hours` INT        NOT NULL DEFAULT 72 COMMENT '免费改期剩余小时阈值',
  `reschedule_fee_rate` DECIMAL(5,2) NOT NULL DEFAULT 20.00 COMMENT '改期调度费率(%)',
  `reschedule_min_hours` INT         NOT NULL DEFAULT 24 COMMENT '距拍摄不足该小时数不可改期',
  `select_deadline_hours` INT        NOT NULL DEFAULT 72 COMMENT '上传样片后选片截止(小时)',
  `retain_days`         INT          NOT NULL DEFAULT 30 COMMENT '未选原片保留天数',
  `faq`                 TEXT         COMMENT '常见问题(JSON数组: [{q,a}])',
  `service_flow`        TEXT         COMMENT '服务流程(JSON数组)',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_studio_setting_company` (`company_id`, `deleted`),
  KEY `idx_studio_setting_deleted` (`deleted`)
) ENGINE=InnoDB AUTO_INCREMENT=1 DEFAULT CHARSET=utf8 COLLATE=utf8_general_ci COMMENT='工作室设置(预约主页/接单规则)';

-- 登录设备（摄影师 App 账号安全）
DROP TABLE IF EXISTS `sys_user_device`;
CREATE TABLE `sys_user_device` (
  `id`              BIGINT       NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `created_by`      BIGINT       NOT NULL DEFAULT 0 COMMENT '创建人',
  `created_at`      DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `updated_by`      BIGINT       NOT NULL DEFAULT 0 COMMENT '修改人',
  `updated_at`      DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '修改时间',
  `deleted`         BIGINT       NOT NULL DEFAULT 0 COMMENT '是否删除 0-否 1-是',
  `company_id`      BIGINT       NOT NULL DEFAULT 0 COMMENT '公司ID',
  `user_id`         BIGINT       NOT NULL DEFAULT 0 COMMENT '用户ID',
  `device_name`     VARCHAR(100) DEFAULT NULL COMMENT '设备名称(如 iPhone 15)',
  `platform`        VARCHAR(20)  DEFAULT NULL COMMENT '平台 ios/android/pc/wechat',
  `last_ip`         VARCHAR(50)  DEFAULT NULL COMMENT '最近登录IP',
  `last_active_at`  DATETIME     DEFAULT NULL COMMENT '最近活跃时间',
  `status`          TINYINT      NOT NULL DEFAULT 1 COMMENT '状态 1-正常 0-已踢出',
  PRIMARY KEY (`id`),
  KEY `idx_device_user` (`user_id`),
  KEY `idx_device_company` (`company_id`),
  KEY `idx_device_deleted` (`deleted`)
) ENGINE=InnoDB AUTO_INCREMENT=1 DEFAULT CHARSET=utf8 COLLATE=utf8_general_ci COMMENT='登录设备';
