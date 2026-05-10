-- Roll back analytics views and helper function.
-- Drop in reverse dependency order.

DROP FUNCTION IF EXISTS refresh_analytics_views();

DROP VIEW IF EXISTS v_hospital_tool_usage;

DROP MATERIALIZED VIEW IF EXISTS mv_platform_daily;
DROP MATERIALIZED VIEW IF EXISTS mv_hospital_top_conditions;
DROP MATERIALIZED VIEW IF EXISTS mv_hospital_call_daily;
