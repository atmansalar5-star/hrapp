// پیکربندی محیطی سرور: آدرس، شرکت فعال و کلیدها از متغیر محیطی/فایل .env.

package main

import (
	"bufio"
	"os"
	"strings"
)

// loadDotEnv متغیرهای محیطی را از یک فایل سادهٔ KEY=VALUE می‌خواند.
//
// این کار در خود سرور انجام می‌شود (نه در اسکریپت پوسته) تا روی ویندوز و لینوکس یک
// رفتار داشته باشد و بالا آوردن محیط واقعی با یک دستور ممکن باشد. دو قاعدهٔ عمدی:
//
//   - متغیری که از قبل در محیط تعریف شده باشد بازنویسی نمی‌شود؛ اولویت همیشه با
//     محیط استقرار است (همان چیزی که در کانتینر یا systemd انتظار می‌رود).
//   - نبودن فایل خطا نیست؛ یعنی «فقط از محیط بخوان».
func loadDotEnv(path string) error {
	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, found := strings.Cut(line, "=")
		if !found {
			continue
		}
		key = strings.TrimSpace(key)
		if key == "" {
			continue
		}
		value = strings.Trim(strings.TrimSpace(value), `"'`)
		if _, exists := os.LookupEnv(key); !exists {
			_ = os.Setenv(key, value)
		}
	}
	return scanner.Err()
}
