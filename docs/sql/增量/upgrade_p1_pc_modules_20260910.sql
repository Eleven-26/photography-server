-- =====================================================================
-- SLOT 摄影工作室管理系统 增量升级脚本（P1 批次 C：交付工作台 / 工作室设置）
-- 依据：原型《SLOT_PC后台管理》——「选片与精修」新建交付任务表单、「工作室设置」基础信息
-- 日期：2026-09-10
-- 内容：
--   1. sys_company   新增 city（所在城市）、intro（工作室简介）      —— P1-11
--   2. biz_delivery  新增 raw_count（原片数量计划）、retouch_target（计划精修张数） —— P1-8
-- 执行方式：在已有库上执行；新增字段均为可空或带默认值，**无需回填存量数据**。
--           不保证幂等（字段已存在会报错，属预期）。
-- =====================================================================

USE `photography`;
SET NAMES utf8mb4;

-- ---------------------------------------------------------------------
-- 一、公司：所在城市 / 工作室简介（设置页「基础信息」对外展示字段）
-- ---------------------------------------------------------------------
ALTER TABLE `sys_company`
  ADD COLUMN `city`  VARCHAR(50)   DEFAULT NULL COMMENT '所在城市'   AFTER `logo`,
  ADD COLUMN `intro` VARCHAR(1000) DEFAULT NULL COMMENT '工作室简介' AFTER `city`;

-- ---------------------------------------------------------------------
-- 二、交付单：计划口径字段（新建交付任务时录入，用于看板进度展示）
--     与 sample_count(实际样片数) / retouched_count(实际完成数) 区分：
--     这两个是「计划值」，进度 = 实际 / 计划。
-- ---------------------------------------------------------------------
ALTER TABLE `biz_delivery`
  ADD COLUMN `raw_count`      INT NOT NULL DEFAULT 0 COMMENT '原片数量(计划)'   AFTER `stage`,
  ADD COLUMN `retouch_target` INT NOT NULL DEFAULT 0 COMMENT '计划精修张数'     AFTER `raw_count`;
