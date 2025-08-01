# alert

Created by: Назар Парносов

# Monitoring & Alerts

## 1. High Error Rates (5xx)

**Metric**: `*_http_requests_total` with `status_class="5xx"`

- **Alert**: Trigger when the 5xx error rate exceeds **5%** of total HTTP requests over a 5-minute window.
- **Severity**: Critical

**Rationale**: A spike in server errors indicates systemic failures (e.g., database connectivity, panic recovery) that degrade all client requests. Early detection prevents prolonged downtime.

---

## 2. Elevated Client Errors (4xx)

**Metric**: `*_http_requests_total` with `status_class="4xx"`

- **Alert**: Trigger when the 4xx client error rate exceeds **10%** of total HTTP requests over 10 minutes.
- **Severity**: Warning

**Rationale**: A surge in bad requests may point to breaking API contract changes or misbehaving clients. It's less severe than 5xx but warrants investigation to avoid user frustration.

---

5. RabbitMQ Publish/Consume Failures

**Metrics**:

- `_rabbitmq_publish_total` with `result="error"`
- `consumer_errors_total` (notification service)
- **Alert (Publish)**: Trigger when publish error ratio > **1%** over 10 minutes.
- **Alert (Consume)**: Trigger when `consumer_errors_total` increases by **5** over 5 minutes.
- **Severity**: Critical

**Rationale**: Messaging failures break critical workflows (email confirmation, weather notifications). Monitoring both sides ensures message flow continuity.

---

## 6. Cron Job Failures & Duration

**Metrics**:

- `cron_runs_total` failures inferred via technical errors
- `cron_run_duration_seconds`
- **Alert**: Trigger when a cron job does not complete within **2x** its expected schedule interval (e.g., hourly job > 2h) or when `cron_runs_total` for a frequency stops incrementing.
- **Severity**: Critical

**Rationale**: Notifier cron jobs driving weather emails must run on schedule. Missed or hung jobs delay user notifications.

---

## 7. Email Send Failures

**Metrics**:

- `email_sent_total`
- `email_errors_total`
- **Alert**: Trigger when `email_errors_total` / `email_sent_total` > **2%** over 15 minutes.
- **Severity**: Critical

**Rationale**: High email failure rates indicate SMTP issues, potentially leaving users without confirmations or weather updates.

---

## 8. Service Uptime

**Metric**: `*_service_uptime_seconds`

- **Alert**: Trigger if uptime gauge resets (drops to near zero), indicating a restart.
- **Severity**: Info / Warning

**Rationale**: Unexpected restarts may reflect crashes or deploy loops. Correlate with log crashes to diagnose root cause.

---

## 9. In-Flight HTTP Requests

**Metric**: `http_requests_in_flight`

- **Alert**: Trigger when in-flight requests exceed **100**.
- **Severity**: Warning

**Rationale**: High concurrency may signal a surge in traffic or slow downstream dependencies, risking resource exhaustion.

---

### Summary

By combining structured logs (for contextual failure analysis) with RED metrics (Rate, Errors, Duration), we can set **actionable alerts** that protect our SLIs and ensure a reliable subscription and notification experience.