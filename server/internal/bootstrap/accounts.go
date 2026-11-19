// انتخاب حساب‌های کاربری در استقرار: فایل کاربران عملیاتی یا حساب‌های نمونهٔ توسعه.

package bootstrap

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"hrms/internal/platform/auth"
	"hrms/internal/platform/localstore"
)

// operationalAccounts حساب‌های ورود سامانه را برمی‌گرداند.
//
// قاعدهٔ استقرار (بند OPS-03 اسپک عملیاتی): پایگاه‌دادهٔ واقعی با حساب نمایشی
// (`admin`/`demo`) بالا نمی‌آید. حساب‌های عملیاتی در فایل JSON کنار پوشهٔ داده
// نگه‌داری می‌شوند: رمزها Argon2id درهم‌شده‌اند و پرونده با دسترسی محدود نوشته می‌شود.
// اگر سرور روی پایگاه‌داده بالا آمده و فایل کاربران نباشد، در نخستین اجرا ساخته می‌شود
// و رمز موقت تصادفی مدیر یک‌بار در لاگ اعلام می‌شود.
func operationalAccounts(cfg Config, dataDir string, logger *slog.Logger) ([]auth.Account, error) {
	path := strings.TrimSpace(cfg.UsersFile)
	hasDatabase := strings.TrimSpace(cfg.DatabaseURL) != ""
	if path == "" {
		// بدون پایگاه‌دادهٔ واقعی (آزمون/نمایش) یا با اجازهٔ صریح نمایش، حساب‌های نمونه.
		if !hasDatabase || cfg.AllowDemoAccounts {
			return auth.SeedAccounts()
		}
		// پیش‌فرض عملیاتی: فایل کاربران کنار پوشهٔ داده (پاک‌شدنی با به‌روزرسانی برنامه نیست).
		path = filepath.Join(dataDir, "users.json")
	}

	password := cfg.AdminPassword
	if strings.TrimSpace(password) == "" {
		if _, statErr := os.Stat(path); os.IsNotExist(statErr) {
			if password, _ = localstore.RandomPassword(); password == "" {
				return nil, errors.New("تولید رمز موقت مدیر ناموفق بود")
			}
			user := strings.TrimSpace(cfg.AdminUser)
			if user == "" {
				user = localstore.DefaultAdminUser
			}
			// رمز موقت یک‌بار اعلام می‌شود؛ همین خط تنها جای آن است و باید در نخستین
			// ورود عوض شود (سند عملیاتی، بخش «اولین ورود»).
			logger.Warn("فایل کاربران تازه ساخته شد؛ رمز موقت مدیر را یادداشت و همان روز عوض کنید",
				"file", path, "user", user, "password", password)
		}
	}

	accounts, created, err := localstore.Users(path, cfg.AdminUser, password)
	if err != nil {
		return nil, fmt.Errorf("فایل کاربران عملیاتی «%s»: %w", path, err)
	}
	if created {
		logger.Info("فایل کاربران عملیاتی ساخته شد", "file", path, "accounts", len(accounts))
	} else {
		logger.Info("کاربران از فایل عملیاتی خوانده شدند", "file", path, "accounts", len(accounts))
	}
	return accounts, nil
}
