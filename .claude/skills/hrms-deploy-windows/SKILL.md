---
name: hrms-deploy-windows
description: راه‌اندازی و عیب‌یابی سرور HRMS روی این ویندوز — سرویس PostgreSQL، تسک زمان‌بندی‌شدهٔ «HRMS Server» روی 127.0.0.1:8099، بازسازی باینری، و خطاهای رایج (بالا نبودن سرور، دام باتری، پورت اشغال). وقتی کاربر می‌گوید «سرور بالا نیست»، «صفحه باز نمی‌شود»، «ری‌استارت کن»، «استقرار/نصب»، این اسکیل را بار کنید.
metadata:
  project: hrapp
  area: ops
---

# استقرار و عیب‌یابی روی ویندوز

## گام صفر: `/readyz`

پیش از هر عیب‌یابی رابط، وضعیت سرور را ببین. «صفحه باز نمی‌شود» در بیشتر دوره‌ها یعنی
**سرور بالا نیست**، نه اینکه کد خراب است.

```powershell
try { (Invoke-WebRequest http://127.0.0.1:8099/readyz -TimeoutSec 4 -UseBasicParsing).StatusCode } catch { $_.Exception.Message }
Get-Process hrmsd -ErrorAction SilentlyContinue
Get-ScheduledTask -TaskName 'HRMS Server' | Select-Object State
Get-Service HRMS-PostgreSQL | Select-Object Status, StartType
```

## دو مؤلفهٔ خودکار

- **سرویس `HRMS-PostgreSQL`**: خوشهٔ محلی روی `127.0.0.1:5432`، شروع خودکار هنگام بوت، حساب
  `NT AUTHORITY\NetworkService`. کنترلش دسترسی **مدیر** می‌خواهد.
- **تسک `HRMS Server`**: محرکِ شروع سیستم، حساب `SYSTEM`، تأخیر ۴۰ ثانیه؛
  `server\bin\hrmsd.exe` را روی `127.0.0.1:8099` اجرا می‌کند. خروجی: `server\server.log`.

گزارش وضعیت و ابزار دستی:

```powershell
powershell -NoProfile -ExecutionPolicy Bypass -File "$env:USERPROFILE\tools\hrms-stack-start.ps1" -Status
```

## چرخهٔ کار توسعه

```powershell
Stop-ScheduledTask -TaskName 'HRMS Server'      # آزادکردن پورت
cd D:\hrapp
start-HRMS.bat                                   # ساخت از سورس و اجرا
# ... کار ...
Start-ScheduledTask -TaskName 'HRMS Server'      # بازگرداندن به حالت سرویس
```

`restart-HRMS.cmd` در ریشهٔ مخزن هم همان چرخه را کوتاه می‌کند. پس از هر تغییر کد، باینری
`server\bin\hrmsd.exe` باید تازه شود، وگرنه مسیرهای تازه در `:8099` نیستند.

## دام باتری لپ‌تاپ (دیده‌شده در v2.10)

اگر تسک با تنظیم‌های پیش‌فرض ساخته شده باشد (`DisallowStartIfOnBatteries = True`)، با رفتن
لپ‌تاپ روی باتری ویندوز سرور را «خاموشی نرم» می‌دهد و شروع دوباره را هم نمی‌دهد: `/readyz`
جواب نمی‌دهد، `Get-ScheduledTask` وضعیت `Queued` نشان می‌دهد و `Get-Process hrmsd` خالی است،
در حالی که PostgreSQL سالم است. رفع دائمی نیازمند یک‌بار PowerShell با دسترسی **مدیر**:

```powershell
$t = Get-ScheduledTask -TaskName 'HRMS Server'
$t.Settings.DisallowStartIfOnBatteries = $false
$t.Settings.StopIfGoingOnBatteries = $false
Set-ScheduledTask -TaskName 'HRMS Server' -Settings $t.Settings
```

بدون دسترسی مدیر، راه موقت اجرای همان فرمان در نشست کاربر است:

```powershell
cd D:\hrapp\server; .\bin\hrmsd.exe -addr 127.0.0.1:8099 -org bandar_imam -web "..\design\prototype" -root . -env .env -data data
```

## قاعده

هر تغییری که سرویس/تسک را متوقف یا بازراه‌اندازی می‌کند **نیازمند اجازهٔ صریح کاربر** است؛
خودسرانه سرویس تولیدی را پایین نیاور. اگر اجازه نبود، فقط بگو چه کاری لازم است و چه چیزی
تا آن لحظه در دسترس نیست (مثلاً مسیرهای تازهٔ API فقط پس از بازسازی سرو می‌شوند).