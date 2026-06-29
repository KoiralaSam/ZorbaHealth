DROP FUNCTION IF EXISTS analytics.refresh_safe_views();
DROP VIEW IF EXISTS analytics.patient_call_history_v;
DROP MATERIALIZED VIEW IF EXISTS analytics.platform_daily_mv;
DROP VIEW IF EXISTS analytics.hospital_recent_activity_v;
DROP VIEW IF EXISTS analytics.hospital_tool_usage_v;
DROP MATERIALIZED VIEW IF EXISTS analytics.hospital_top_conditions_mv;
DROP MATERIALIZED VIEW IF EXISTS analytics.hospital_call_daily_mv;
DROP SCHEMA IF EXISTS analytics;
