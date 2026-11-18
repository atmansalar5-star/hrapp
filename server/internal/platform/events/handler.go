// مسیرهای مدیریتی صندوق خروجی رخداد.
//
// این مسیرها فقط برای مدیر سیستم است: فهرست رخدادها نام کاربران و مسیرهای درخواست‌ها
// را نشان می‌دهد و «بازگرداندن به صف» یک عمل مدیریتی است. هیچ قاعدهٔ کسب‌وکاری اینجا
// نیست؛ فقط خواندن وضعیت صندوق و بازگرداندن یک ردیف.

package events

import (
	"encoding/json"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"hrms/internal/contracts"
	"hrms/internal/platform/auth"
	"hrms/internal/platform/httpx"
)

// ConfigHandler وضعیت صندوق و تنظیمات جاری ارسال را برمی‌گرداند.
func ConfigHandler(bus *Bus) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !allowAdmin(w, r) {
			return
		}
		counts := map[string]int{}
		if bus != nil && bus.Store() != nil {
			if c, err := bus.Stats(r.Context()); err == nil {
				counts = c
			}
		}
		httpx.WriteJSON(w, http.StatusOK, map[string]any{
			"enabled":         bus.Enabled(),
			"target":          maskedURL(bus.URL()),
			"secretSet":       bus != nil && bus.cfg.Secret != "",
			"intervalSeconds": int(bus.Interval().Seconds()),
			"maxAttempts":     bus.MaxAttempts(),
			"counts":          counts,
		})
	}
}

// OutboxHandler آخرین رخدادهای صندوق را برمی‌گرداند.
func OutboxHandler(bus *Bus) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !allowAdmin(w, r) {
			return
		}
		if bus == nil || bus.Store() == nil {
			httpx.WriteJSON(w, http.StatusOK, map[string]any{"items": []any{}, "total": 0})
			return
		}
		status := strings.TrimSpace(r.URL.Query().Get("status"))
		limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
		if limit <= 0 || limit > 500 {
			limit = 50
		}
		rows, err := bus.Store().Recent(r.Context(), status, limit)
		if err != nil {
			httpx.WriteErrorMsg(w, http.StatusInternalServerError, contracts.CodeInternal, err.Error(), nil)
			return
		}
		items := make([]map[string]any, 0, len(rows))
		for _, row := range rows {
			items = append(items, deliveryJSON(row))
		}
		httpx.WriteJSON(w, http.StatusOK, map[string]any{"items": items, "total": len(items)})
	}
}

// RetryHandler یک ردیف را برای تلاش دوباره به صف برمی‌گرداند.
func RetryHandler(bus *Bus) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !allowAdmin(w, r) {
			return
		}
		if bus == nil || bus.Store() == nil {
			httpx.WriteErrorMsg(w, http.StatusConflict, contracts.CodeConflict, "صندوق خروجی رخداد فعال نیست.", nil)
			return
		}
		id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
		if err != nil {
			httpx.WriteErrorMsg(w, http.StatusBadRequest, contracts.CodeBadRequest, "شناسهٔ رخداد نامعتبر است.", nil)
			return
		}
		if err := bus.Store().Requeue(r.Context(), id, time.Now()); err != nil {
			httpx.WriteError(w, http.StatusNotFound, contracts.CodeNotFound, nil)
			return
		}
		httpx.WriteJSON(w, http.StatusOK, map[string]any{"id": id, "status": StatusPending})
	}
}

// deliveryJSON یک ردیف صندوق را به شکل API تبدیل می‌کند.
func deliveryJSON(row Delivery) map[string]any {
	out := map[string]any{
		"id":             row.ID,
		"eventId":        row.Event.ID,
		"name":           row.Event.Name,
		"occurredAt":     row.Event.OccurredAt.UTC().Format(time.RFC3339),
		"actor":          row.Event.Actor,
		"role":           row.Event.Role,
		"module":         row.Event.Module,
		"method":         row.Event.Method,
		"path":           row.Event.Path,
		"httpStatus":     row.Event.Status,
		"subject":        row.Event.Subject,
		"attempts":       row.Attempts,
		"deliveryStatus": row.Status(),
	}
	if !row.NextAttemptAt.IsZero() {
		out["nextAttemptAt"] = row.NextAttemptAt.UTC().Format(time.RFC3339)
	}
	if row.DeliveredAt != nil {
		out["deliveredAt"] = row.DeliveredAt.UTC().Format(time.RFC3339)
	}
	if row.LastError != "" {
		out["lastError"] = row.LastError
	}
	if payload, err := json.Marshal(row.Event.Payload); err == nil {
		out["payload"] = json.RawMessage(payload)
	}
	return out
}

// maskedURL نشانی مقصد را به شکل «مبدأ + مسیر» برمی‌گرداند (بدون اطلاعات ورود اگر در
// نشانی باشد).
func maskedURL(raw string) string {
	if raw == "" {
		return ""
	}
	parsed, err := url.Parse(raw)
	if err != nil {
		return "(نشانی نامعتبر)"
	}
	clean := *parsed
	clean.User = nil
	return clean.String()
}

// allowAdmin فقط مدیر سیستم را می‌پذیرد.
func allowAdmin(w http.ResponseWriter, r *http.Request) bool {
	user, ok := auth.UserFrom(r.Context())
	if !ok {
		httpx.WriteError(w, http.StatusUnauthorized, contracts.CodeUnauthorized, nil)
		return false
	}
	if user.Role != contracts.RoleAdmin {
		httpx.WriteError(w, http.StatusForbidden, contracts.CodeForbidden, nil)
		return false
	}
	return true
}