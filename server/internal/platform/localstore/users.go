// Package localstore انبار محلی اجرای مستقل روی یک رایانه.
//
// چرا جدا از ماژول‌ها: اجرای مستقل یک زیرسامانه (مثل «سیستم پاداش») نباید به
// پایگاه‌داده، به ماژول تنظیمات یا به هیچ سرویس بیرونی وابسته باشد. این بسته فقط سه
// چیز را در یک پوشهٔ داده نگه می‌دارد و همه‌چیز در یک فایل خوانا و قابل پشتیبان‌گیری
// خلاصه می‌شود:
//
//	users.json  کاربران و رمز درهم‌شده (هیچ رمزی به‌صورت متن ساده ذخیره نمی‌شود)
//	secret.key  کلید امضای نشست، تا با هر راه‌اندازی دوباره، ورود کاربران باطل نشود
//	staff.csv   فهرست کارکنان (کد پرسنلی، نام، واحد، سمت، وضعیت) — قابل ویرایش با Excel
//
// خودِ رکوردهای گردش‌کاری در انبار فایل JSON موتور recordkit می‌نشینند
// (recordkit.NewFileStore) تا منطق فیلتر و صفحه‌بندی یک نسخهٔ دوم نداشته باشد.
package localstore

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"hrms/internal/contracts"
	"hrms/internal/platform/auth"
)

// UserAccount یک کاربر در فایل users.json.
//
// رمز فقط به‌صورت درهم‌شده (bcrypt، همان تابع ماژول احراز هویت) نگه داشته می‌شود؛ پس
// فایل کاربران حتی اگر دست کسی بیفتد، رمز را لو نمی‌دهد.
type UserAccount struct {
	Username     string `json:"username"`
	DisplayName  string `json:"displayName,omitempty"`
	Role         string `json:"role"`
	EmployeeID   string `json:"employeeId,omitempty"`
	DepartmentID string `json:"departmentId,omitempty"`
	PasswordHash string `json:"passwordHash"`
	Disabled     bool   `json:"disabled,omitempty"`
}

// usersFile شکل فایل کاربران.
type usersFile struct {
	Note  string        `json:"note,omitempty"`
	Users []UserAccount `json:"users"`
}

// DefaultAdminUser کاربر پیش‌فرض اجرای نخست.
const DefaultAdminUser = "admin"

// DefaultAdminPassword رمز پیش‌فرض اجرای نخست؛ در راه‌اندازی با صدای بلند هشدار داده
// می‌شود تا کاربر آن را عوض کند (با پرچم -admin-password یا ویرایش همین فایل).
const DefaultAdminPassword = "6468"

const usersNote = "کاربران سیستم پاداش. رمزها درهم‌شده‌اند؛ برای تغییر رمز، سامانه را با پرچم -admin-password بالا بیاورید."

// Users فایل کاربران را می‌خواند یا در اجرای نخست می‌سازد و به شکل حساب‌های ماژول
// احراز هویت برمی‌گرداند.
//
// اگر bootstrapPassword ناتهی باشد، همان کاربر (bootstrapUser یا کاربر پیش‌فرض) با رمز
// تازه به‌روزرسانی می‌شود؛ این راه، بازنشانی رمز بدون ویرایش دستی فایل است. تغییر روی
// دیسک ذخیره می‌شود تا اجرای بعدی همان رمز را بپذیرد.
func Users(path, bootstrapUser, bootstrapPassword string) (accounts []auth.Account, created bool, err error) {
	if strings.TrimSpace(path) == "" {
		return nil, false, fmt.Errorf("مسیر فایل کاربران خالی است")
	}
	if dir := filepath.Dir(path); dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, false, fmt.Errorf("ساخت پوشهٔ داده ناموفق: %w", err)
		}
	}

	file := usersFile{Note: usersNote}
	raw, readErr := os.ReadFile(path)
	switch {
	case readErr == nil && len(raw) > 0:
		if err := json.Unmarshal(raw, &file); err != nil {
			return nil, false, fmt.Errorf("فایل کاربران «%s» خوانا نیست: %w", path, err)
		}
	case readErr == nil || os.IsNotExist(readErr):
		// اجرای نخست: یک کاربر مدیر ساخته می‌شود تا سامانه از همان ابتدا قابل ورود باشد.
		user := firstNonEmpty(bootstrapUser, DefaultAdminUser)
		password := firstNonEmpty(bootstrapPassword, DefaultAdminPassword)
		hash, err := auth.HashPassword(password)
		if err != nil {
			return nil, false, fmt.Errorf("درهم‌سازی رمز ناموفق: %w", err)
		}
		file.Users = []UserAccount{{
			Username: user, DisplayName: "مدیر پاداش", Role: string(contracts.RoleAdmin),
			PasswordHash: hash,
		}}
		created = true
		if err := saveUsers(path, file); err != nil {
			return nil, false, err
		}
	default:
		return nil, false, fmt.Errorf("خواندن فایل کاربران ناموفق: %w", readErr)
	}

	// بازنشانی/ساخت رمز با پرچم اجرا.
	if strings.TrimSpace(bootstrapPassword) != "" {
		if err := upsertPassword(path, &file, firstNonEmpty(bootstrapUser, DefaultAdminUser), bootstrapPassword); err != nil {
			return nil, false, err
		}
	}

	for _, u := range file.Users {
		if strings.TrimSpace(u.Username) == "" || u.PasswordHash == "" {
			continue
		}
		accounts = append(accounts, auth.Account{
			Username:     u.Username,
			PasswordHash: u.PasswordHash,
			Disabled:     u.Disabled,
			User: contracts.User{
				Username:     u.Username,
				DisplayName:  firstNonEmpty(u.DisplayName, u.Username),
				Role:         role(u.Role),
				EmployeeID:   u.EmployeeID,
				DepartmentID: u.DepartmentID,
			},
		})
	}
	if len(accounts) == 0 {
		return nil, created, fmt.Errorf("فایل کاربران «%s» هیچ کاربر فعالی ندارد", path)
	}
	return accounts, created, nil
}

// upsertPassword رمز یک کاربر را می‌سازد یا عوض می‌کند و فایل را ذخیره می‌کند.
//
// سیاست قدرت رمز همین‌جا اعمال می‌شود: این مسیر همان جایی است که راهبر رمز دلخواه
// می‌گذارد (پرچم بازنشانی)، پس ضعیف‌بودنش باید همان‌جا و با پیام روشن رد شود — نه
// اینکه در استقرار کشف شود.
func upsertPassword(path string, file *usersFile, username, password string) error {
	if err := auth.ValidateStrength(password); err != nil {
		return err
	}
	hash, err := auth.HashPassword(password)
	if err != nil {
		return fmt.Errorf("درهم‌سازی رمز ناموفق: %w", err)
	}
	for i := range file.Users {
		if file.Users[i].Username == username {
			file.Users[i].PasswordHash = hash
			file.Users[i].Disabled = false
			return saveUsers(path, *file)
		}
	}
	file.Users = append(file.Users, UserAccount{
		Username: username, DisplayName: "مدیر پاداش",
		Role: string(contracts.RoleAdmin), PasswordHash: hash,
	})
	return saveUsers(path, *file)
}

func saveUsers(path string, file usersFile) error {
	raw, err := json.MarshalIndent(file, "", "  ")
	if err != nil {
		return fmt.Errorf("سریال‌سازی کاربران ناموفق: %w", err)
	}
	raw = append(raw, '\n')
	if err := os.WriteFile(path, raw, 0o600); err != nil {
		return fmt.Errorf("ذخیرهٔ فایل کاربران ناموفق: %w", err)
	}
	return nil
}

// Secret کلید امضای نشست را از فایل می‌خواند یا در اجرای نخست می‌سازد.
//
// چرا در فایل و نه تصادفی در هر اجرا: اگر کلید هر بار عوض شود، با هر راه‌اندازی همهٔ
// نشست‌ها باطل می‌شوند و کاربر باید دوباره وارد شود. یک زیرسامانهٔ مستقل روی یک رایانه
// باید کلید پایدار داشته باشد، ولی همچنان کلید در کد و در مخزن نمی‌نشیند.
func Secret(dataDir string) ([]byte, error) {
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		return nil, fmt.Errorf("ساخت پوشهٔ داده ناموفق: %w", err)
	}
	path := filepath.Join(dataDir, "secret.key")
	if raw, err := os.ReadFile(path); err == nil && len(strings.TrimSpace(string(raw))) >= 16 {
		return []byte(strings.TrimSpace(string(raw))), nil
	}
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return nil, fmt.Errorf("ساخت کلید تصادفی ناموفق: %w", err)
	}
	key := base64.RawURLEncoding.EncodeToString(buf)
	if err := os.WriteFile(path, []byte(key+"\n"), 0o600); err != nil {
		return nil, fmt.Errorf("ذخیرهٔ کلید امضا ناموفق: %w", err)
	}
	return []byte(key), nil
}

// role نقش کاربر را به یکی از نقش‌های شناخته‌شدهٔ سامانه نگاشت می‌کند؛ مقدار ناشناخته
// به نقش کارمند برمی‌گردد تا یک اشتباه تایپی در فایل، دسترسی مدیریتی نسازد.
func role(value string) contracts.Role {
	r := contracts.Role(strings.TrimSpace(value))
	if _, ok := contracts.RoleFa[r]; ok {
		return r
	}
	return contracts.RoleEmployee
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}
