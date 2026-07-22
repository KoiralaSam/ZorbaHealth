-- No-op bridge version after welfare migrations were renumbered from 000016/000017
-- to 000018/000019 to make room for backend-contracts 000009-000016.
-- Keeps golang-migrate happy when a database already recorded version 17.
SELECT 1;
