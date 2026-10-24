//go:build windows

package main

import (
	"errors"
	"fmt"
	"log/slog"
	"path/filepath"

	webview2 "github.com/jchv/go-webview2"
)

// errNoWebView2 وقتی موتور WebView2 روی این ویندوز نصب نیست.
var errNoWebView2 = errors.New("موتور Microsoft Edge WebView2 روی این ویندوز نصب نیست")

// windowConfig پیکربندی پنجرهٔ بومی برنامه.
type windowConfig struct {
	Title   string
	URL     string
	DataDir string
	BaseDir string
	Port    int
	Debug   bool
	Logger  *slog.Logger
}

// windowWidth و windowHeight اندازهٔ آغازین پنجره؛ برای فرم‌های جدولی و داشبورد کافی است.
const (
	windowWidth  = 1200
	windowHeight = 840
)

// runWindow پنجرهٔ بومی را می‌سازد، پل داخلی را به رابط وصل می‌کند و تا بسته شدن پنجره
// منتظر می‌ماند. بازگشت از این تابع یعنی کاربر پنجره را بسته است (یا ساخت ناموفق بوده).
func runWindow(cfg windowConfig) error {
	// پیش از ساخت پنجره: وضوح متن روی نمایشگرهای با مقیاس بزرگ (۱۲۵٪ به بالا).
	enableHighDPI()

	view := webview2.NewWithOptions(webview2.WebViewOptions{
		Debug:     cfg.Debug,
		AutoFocus: true,
		// پروفایل داخلی موتور هم داخل پوشهٔ داده می‌نشیند تا کل برنامه یک پوشه باشد
		// و روی رایانهٔ کاربر هیچ ردی بیرون از آن نماند.
		DataPath: filepath.Join(cfg.DataDir, "webview2"),
		WindowOptions: webview2.WindowOptions{
			Title:  cfg.Title,
			Width:  windowWidth,
			Height: windowHeight,
			Center: true,
		},
	})
	if view == nil {
		return errNoWebView2
	}
	defer view.Destroy()

	view.SetSize(windowWidth, windowHeight, webview2.HintNone)

	if err := bindDesktopBridge(view, cfg); err != nil {
		return err
	}

	view.Navigate(cfg.URL)
	view.Run()
	return nil
}

// bindDesktopBridge توابع بومی را به‌عنوان توابع سراسری به جاوااسکریپت وصل می‌کند.
//
// چرا این پل لازم است: رابط وب نمی‌تواند پوشه باز کند یا فایل بسازد. در نسخهٔ وب،
// کاربر خودش پوشه را کپی می‌کند؛ در نسخهٔ دسکتاپ دکمه نشان داده می‌شود و همین توابع
// کار را انجام می‌دهند. رابط این توابع را با تشخیص وجودشان پیدا می‌کند، پس همان فایل
// HTML بدون تغییر بین دو نسخه کار می‌کند.
func bindDesktopBridge(view webview2.WebView, cfg windowConfig) error {
	bindings := []struct {
		name string
		fn   any
	}{
		{"desktopInfo", func() (map[string]any, error) {
			return map[string]any{
				"title": appTitleFa, "version": appVersion,
				"dataDir": cfg.DataDir, "port": cfg.Port, "url": cfg.URL,
			}, nil
		}},
		{"desktopOpenDataFolder", func() error {
			return openFolder(cfg.DataDir)
		}},
		// desktopReady را خودِ صفحه پس از بارشدن کامل صدا می‌زند. چرا: برنامه بدون
		// کنسول است، پس تنها شاهد اینکه «پنجره واقعاً رابط را نشان داد» همین گزارش است
		// — برای پشتیبانی هم به کار می‌آید (لاگ می‌گوید کدام نسخه و چند کارمند/رکورد
		// به کاربر نمایش داده شد).
		{"desktopReady", func(payload string) error {
			cfg.Logger.Info("رابط در پنجرهٔ دسکتاپ بار شد", "ui", payload)
			return nil
		}},
		{"desktopBackup", func() (map[string]any, error) {
			result, err := backupData(cfg.DataDir, filepath.Join(cfg.BaseDir, "backup"))
			if err != nil {
				return nil, err
			}
			return map[string]any{
				"path": result.Path, "files": result.Files, "bytes": result.Bytes,
			}, nil
		}},
	}
	for _, binding := range bindings {
		if err := view.Bind(binding.name, binding.fn); err != nil {
			return fmt.Errorf("اتصال «%s» به رابط ناموفق: %w", binding.name, err)
		}
	}
	return nil
}
