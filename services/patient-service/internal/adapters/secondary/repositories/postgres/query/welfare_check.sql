-- Welfare-check queries are implemented directly in the repository for row-locking
-- and nullable timestamp ergonomics. This file keeps sqlc aware of the table.
-- name: CountWelfareCheckRequests :one
SELECT COUNT(*) FROM welfare_check_requests;
