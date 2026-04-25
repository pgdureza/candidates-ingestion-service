-- Metrics table for tracking request/processing lifecycle
CREATE TABLE metrics (
  metric_name TEXT PRIMARY KEY,
  value BIGINT DEFAULT 0
);

-- Initialize counters
INSERT INTO metrics (metric_name, value) VALUES
  ('webhooks_total_request', 0),
  ('webhooks_rate_limited', 0),
  ('webhooks_ingested', 0),
  ('webhooks_rejected', 0),
  ('outbox_written', 0),
  ('outbox_process_attempts', 0),
  ('outbox_publish_success', 0),
  ('outbox_publish_failed', 0),
  ('notification_failed', 0),
  ('outbox_cleaned', 0)
ON CONFLICT (metric_name) DO NOTHING;
