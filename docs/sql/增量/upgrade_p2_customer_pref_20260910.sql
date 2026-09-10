-- =====================================================================
-- SLOT 摄影工作室管理系统 增量升级脚本（P2：客户偏好与通知许可）
-- 依据：原型《SLOT_PC后台管理》——「客户档案」抽屉「客户偏好」区块
--       （客户类型 / 通知许可 / 风格 / 常用场景）
-- 日期：2026-09-10
-- 内容：
--   crm_customer 新增 allow_notifications（通知许可）、prefer_style（偏好风格）、
--                 prefer_scene（常用场景）
-- 说明：
--   * 满意度不走本表新增列 —— 由该客户的订单评价（biz_order_review.rating）派生，
--     避免「手工填一个满意度」与真实评分两套数据打架。
--   * 复购判断同样为派生口径：order_count >= 2 即为复购客户。
-- 执行方式：在已有库上执行；新增字段可空或带默认值，**无需回填存量数据**
--           （存量客户通知许可默认置 1=允许）。
--           不保证幂等（字段已存在会报错，属预期）。
-- =====================================================================

USE `photography`;
SET NAMES utf8mb4;

ALTER TABLE `crm_customer`
  ADD COLUMN `allow_notifications` TINYINT      NOT NULL DEFAULT 1 COMMENT '通知许可 0-不允许 1-允许' AFTER `avatar`,
  ADD COLUMN `prefer_style`        VARCHAR(100) DEFAULT NULL     COMMENT '偏好风格(如 自然·生活感)'   AFTER `allow_notifications`,
  ADD COLUMN `prefer_scene`        VARCHAR(100) DEFAULT NULL     COMMENT '常用场景(如 户外公园)'      AFTER `prefer_style`;

-- ---------------------------------------------------------------------
-- 顺带修正：客户等级默认值与注释须与 enum.CustomerLevel 对齐
-- （1-普通 2-黄金 3-铂金 4-钻石）。原 ddl 误写为 DEFAULT 4 —— 按正确语义
-- 那是「钻石」，未显式指定 level 的插入会默认成最高等级。
-- 仅改列定义，不改动存量数据。
-- ---------------------------------------------------------------------
ALTER TABLE `crm_customer`
  MODIFY COLUMN `level` TINYINT NOT NULL DEFAULT 1 COMMENT '客户等级 1-普通 2-黄金 3-铂金 4-钻石';
