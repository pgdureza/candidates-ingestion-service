-- Add webhooks_duplicate metric counter
INSERT INTO metrics (metric_name, value) VALUES
  ('webhooks_duplicate', 0)
ON CONFLICT (metric_name) DO NOTHING;
