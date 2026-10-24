//go:build windows

package main

import (
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"syscall"
	"unsafe"
)

// فراخوانی مستقیم توابع ویندوز با syscall: برای این چند کار کوچک نیازی به هیچ وابستگی
// بومی یا CGO نیست، و همین چیزی است که ساخت «یک فایل اجرایی بدون پیش‌نیاز» را ممکن می‌کند.
var (
	user32   = syscall.NewLazyDLL("user32.dll")
	kernel32 = syscall.NewLazyDLL("kernel32.dll")

	procMessageBoxW        = user32.NewProc("MessageBoxW")
	procSetProcessDPIAware = user32.NewProc("SetProcessDPIAware")
	procCreateMutexW       = kernel32.NewProc("CreateMutexW")
)

const (
	mbOK            = 0x00000000
	mbIconError     = 0x00000010
	mbSetForeground = 0x00010000
	mbTopmost       = 0x00040000

	errorAlreadyExists = 183
)

// errAlreadyRunning نشانهٔ این است که نمونهٔ دیگری از برنامه همان پوشهٔ داده را گرفته است.
var errAlreadyRunning = errors.New("نمونهٔ دیگری از برنامه در حال اجراست")

// showMessage یک پیام بومی ویندوزی نشان می‌دهد.
//
// چرا لازم است: این برنامه بدون کنسول ساخته می‌شود، پس خطای راه‌اندازی هیچ‌جا دیده
// نمی‌شود. یک پیام بومی تنها راه گفتن «چه شد و چه کار کن» به کاربر است.
func showMessage(title, text string) {
	titlePtr, err := syscall.UTF16PtrFromString(title)
	if err != nil {
		return
	}
	textPtr, err := syscall.UTF16PtrFromString(text)
	if err != nil {
		return
	}
	_, _, _ = procMessageBoxW.Call(
		0,
		uintptr(unsafe.Pointer(textPtr)),
		uintptr(unsafe.Pointer(titlePtr)),
		uintptr(mbOK|mbIconError|mbSetForeground|mbTopmost),
	)
}

// enableHighDPI برنامه را آگاه از مقیاس نمایشگر می‌کند تا متن روی نمایشگرهای ۱۲۵٪ و
// بالاتر تار نشود. اگر این فراخوانی در دسترس نباشد، برنامه بدون آن هم کار می‌کند.
func enableHighDPI() {
	_, _, _ = procSetProcessDPIAware.Call()
}

// acquireSingleInstance یک میوتکس نامدار و پایدار می‌سازد تا دو نمونه روی یک پوشهٔ داده
// بالا نیایند؛ دو نویسندهٔ هم‌زمان می‌توانند فایل رکوردها را خراب کنند.
//
// چرا میوتکس و نه فایل قفل: میوتکس با بسته شدن فرآیند (حتی با کشتن آن) خودکار آزاد
// می‌شود، پس «قفل مرده» باقی نمی‌ماند. نام از مسیر پوشهٔ داده ساخته می‌شود، بنابراین دو
// پوشهٔ دادهٔ متفاوت می‌توانند هم‌زمان باز باشند.
func acquireSingleInstance(dataDir string) error {
	name := fmt.Sprintf(`Local\HRMSBonusApp-%d`, fnv1a(strings.ToLower(dataDir)))
	namePtr, err := syscall.UTF16PtrFromString(name)
	if err != nil {
		return fmt.Errorf("ساخت نام قفل ناموفق: %w", err)
	}
	handle, _, lastErr := procCreateMutexW.Call(0, 0, uintptr(unsafe.Pointer(namePtr)))
	if handle == 0 {
		return fmt.Errorf("ساخت میوتکس ناموفق: %v", lastErr)
	}
	if errors.Is(lastErr, syscall.Errno(errorAlreadyExists)) {
		return errAlreadyRunning
	}
	// دستهٔ میوتکس عمداً باز می‌ماند: با پایان فرآیند خودکار آزاد می‌شود.
	return nil
}

// openFolder پوشه را در پنجرهٔ ویندوز باز می‌کند.
func openFolder(path string) error {
	// explorer حتی وقتی موفق است کد خروج غیرصفر می‌دهد؛ پس معیار، اجرا شدن فرمان است.
	if err := exec.Command("explorer", path).Start(); err != nil {
		return fmt.Errorf("باز کردن پوشه «%s» ناموفق: %w", path, err)
	}
	return nil
}

// fnv1a هش کوتاه و پایدار FNV-1a؛ برای ساختن نام قفل از مسیر داده.
func fnv1a(text string) uint32 {
	const (
		offset = 2166136261
		prime  = 16777619
	)
	hash := uint32(offset)
	for i := 0; i < len(text); i++ {
		hash ^= uint32(text[i])
		hash *= prime
	}
	return hash
}
