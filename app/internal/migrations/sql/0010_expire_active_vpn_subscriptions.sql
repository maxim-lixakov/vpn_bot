UPDATE subscriptions
SET active_until = now()
WHERE status = 'paid'
  AND active_until > now();
