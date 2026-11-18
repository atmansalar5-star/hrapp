// میان‌افزار رخداد: تغییرهای موفقِ API را در صندوق خروجی ثبت می‌کند.
//
// چرا در لایهٔ HTTP و نه در هر ماژول: قاعدهٔ «هیچ قاعدهٔ کسب‌وکاری بیرون از دامنه»
// اینجا هم برقرار است؛ میان‌افزار هیچ تصمیمی نمی‌گیرد و هیچ فیلدی را تغییر نمی‌دهد،
// فقط می‌گوید «چه درخواستی با چه نتیجه‌ای از API گذشت». ثبت پراکنده در ده‌ها ماژول
// سه ایراد داشت: فراموش‌شدن در ماژول‌های تازه، تکرار کد، و نبودِ پوشش یکسان. یک نقطه
// این پوشش را کامل و آزمون‌پذیر می‌کند.
//
// رخداد فقط برای تغییرهای «موفق» (۲xx) و برای روش‌های تغییردهنده ثبت می‌شود؛ خواندن
// (GET) رخداد نیست، درخواست ناموفق هم تغییر نکرده است. دو استثنا هم عمدی است:
//   - مسیرهای نشست (`/api/v1/auth/...`) رخداد کسب‌وکاری نیستند و ثبت نمی‌شوند.
//   - مسیرهای خود صندوق (`/api/v1/events/...`) ثبت نمی‌شوند تا حلقهٔ بی‌پایان نسازند.
//
// حلقهٔ دوم هم بسته است: مصرف‌کنندهٔ بیرونی (مثلاً n8n) که با API کار می‌کند سرآیند
// `X-HRMS-Integration-Origin` را می‌فرستد و درخواست‌هایش رخداد نمی‌سازند.

package events

import (
	"context"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"hrms/internal/platform/auth"
)

// Middleware میان‌افزار ثبت رخداد.
func Middleware(bus *Bus, log *slog.Logger) func(http.Handler) http.Handler {
	if log == nil {
		log = slog.Default()
	}
	return func(next http.Handler) http.Handler {
		if bus == nil || bus.Store() == nil {
			return next
		}
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			started := time.Now()
			rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
			next.ServeHTTP(rec, r)

			event, ok := EventFromRequest(r, rec.status, time.Since(started))
			if !ok {
				bus.Skip()
				return
			}
			// زمینهٔ درخواست پس از پاسخ لغو می‌شود؛ برای نوشتن در صندوق یک زمینهٔ
			// مستقل با مهلت کوتاه می‌سازیم تا رخداد تغییر از دست نرود.
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			bus.Publish(ctx, event)
		})
	}
}

// EventFromRequest از یک درخواست پایان‌یافته رخداد می‌سازد؛ ok=false یعنی این درخواست
// رخداد نیست.
func EventFromRequest(r *http.Request, status int, took time.Duration) (Event, bool) {
	if r == nil {
		return Event{}, false
	}
	if status < 200 || status >= 300 {
		return Event{}, false
	}
	if r.Header.Get(HeaderOrigin) != "" {
		return Event{}, false
	}
	path := r.URL.Path
	if !strings.HasPrefix(path, "/api/v1/") || strings.HasPrefix(path, "/api/v1/auth/") ||
		strings.HasPrefix(path, "/api/v1/events/") {
		return Event{}, false
	}
	name, module, subject := nameFor(r.Method, path)
	if name == "" {
		return Event{}, false
	}

	actor, role := "", ""
	if user, ok := auth.UserFrom(r.Context()); ok {
		actor = user.Username
		role = string(user.Role)
	}
	payload := map[string]any{
		"method":     r.Method,
		"path":       path,
		"durationMs": took.Milliseconds(),
	}
	if q := r.URL.RawQuery; q != "" {
		payload["query"] = q
	}
	return Event{
		Name:    name,
		Actor:   actor,
		Role:    role,
		Module:  module,
		Method:  r.Method,
		Path:    path,
		Status:  status,
		Subject: subject,
		Payload: payload,
	}, true
}

// nameFor نام رخداد، ماژول و شناسهٔ موضوع را از مسیر می‌سازد.
//
// نام‌گذاری ثابت و قابل پیش‌بینی است: `<ماژول>.<کنش>`؛ کنش از روش HTTP می‌آید
// (POST→created، PATCH/PUT→updated، DELETE→deleted) و اگر مسیر به `/action` ختم شود،
// کنش `action` است. مصرف‌کننده می‌تواند یا روی نام سوئیچ کند یا با `path` دقیق‌تر
// تصمیم بگیرد — هر دو در همان رخداد هست.
func nameFor(method, path string) (name, module, subject string) {
	rest := strings.TrimPrefix(path, "/api/v1/")
	rest = strings.Trim(rest, "/")
	if rest == "" {
		return "", "", ""
	}
	parts := strings.Split(rest, "/")
	module = parts[0]

	verb := ""
	switch strings.ToUpper(method) {
	case http.MethodPost:
		verb = "created"
	case http.MethodPatch, http.MethodPut:
		verb = "updated"
	case http.MethodDelete:
		verb = "deleted"
	default:
		return "", "", ""
	}
	if len(parts) > 1 && parts[len(parts)-1] == "action" {
		verb = "action"
	}
	name = module + "." + verb

	// شناسهٔ موضوع: آخرین بخش مسیر که شکل شناسه دارد (EMP-011، PAY-1404-07، 34006).
	if len(parts) > 1 {
		candidate := parts[len(parts)-1]
		if candidate == "action" && len(parts) > 2 {
			candidate = parts[len(parts)-2]
		}
		if looksLikeID(candidate) {
			subject = candidate
		}
	}
	return name, module, subject
}

// looksLikeID آیا رشته شکل شناسهٔ رکورد را دارد (نه نام مجموعه).
func looksLikeID(s string) bool {
	if s == "" || len(s) > 40 {
		return false
	}
	if _, err := strconv.Atoi(s); err == nil {
		return true
	}
	if !strings.Contains(s, "-") {
		return false
	}
	for _, r := range s {
		if !(r >= 'A' && r <= 'Z') && !(r >= 'a' && r <= 'z') && !(r >= '0' && r <= '9') && r != '-' {
			return false
		}
	}
	return true
}

// statusRecorder کد وضعیت پاسخ را نگه می‌دارد و قابلیت‌های اختیاری نویسنده را
// عبور می‌دهد (بدون این، پاسخ‌های جریانی و دانلود فایل می‌شکستند).
type statusRecorder struct {
	http.ResponseWriter
	status int
	wrote  bool
}

func (s *statusRecorder) WriteHeader(code int) {
	if !s.wrote {
		s.status = code
		s.wrote = true
	}
	s.ResponseWriter.WriteHeader(code)
}

func (s *statusRecorder) Write(b []byte) (int, error) {
	if !s.wrote {
		s.wrote = true
	}
	return s.ResponseWriter.Write(b)
}

// Flush پاسخ‌های جریانی را زنده نگه می‌دارد.
func (s *statusRecorder) Flush() {
	if f, ok := s.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

// Unwrap اجازهٔ دسترسی http.ResponseController به نویسندهٔ اصلی را می‌دهد.
func (s *statusRecorder) Unwrap() http.ResponseWriter { return s.ResponseWriter }