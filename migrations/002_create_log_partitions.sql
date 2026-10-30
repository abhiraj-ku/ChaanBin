-- migrations/002_create_log_partitions.sql
CREATE TABLE IF NOT EXISTS app_logs_2026_10
PARTITION OF app_logs
FOR VALUES FROM ('2026-10-01 00:00:00+00')
             TO   ('2026-11-01 00:00:00+00');

CREATE TABLE IF NOT EXISTS app_logs_2026_11
PARTITION OF app_logs
FOR VALUES FROM ('2026-11-01 00:00:00+00')
             TO   ('2026-12-01 00:00:00+00');