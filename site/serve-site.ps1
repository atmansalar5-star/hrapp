# =====================================================================
#  راه‌انداز سایت معرفی «سامانهٔ جامع منابع انسانی — اچ‌آر اپ»
# ---------------------------------------------------------------------
#  چه می‌کند: یک سرور استاتیک محلی روی 127.0.0.1:8098 بالا می‌آورد و پس از
#  دو ثانیه، سایت را در مرورگر پیش‌فرض باز می‌کند. تا وقتی این پنجره باز
#  است سرور زنده می‌ماند؛ برای پایان، Ctrl+C یا بستن پنجره کافی است.
#
#  چرا سرور و نه فایل مستقیم: سایت با file:// هم کار می‌کند، ولی با یک
#  سرور محلی نشانی تمیز و رفتار یکسان در همهٔ مرورگرها داریم.
#  اگر پایتون نصب نباشد، همان فایل index.html مستقیم باز می‌شود.
# =====================================================================
param(
  [int]$Port = 8098,
  [switch]$NoOpen
)

$ErrorActionPreference = 'Stop'
try { [Console]::OutputEncoding = [System.Text.Encoding]::UTF8 } catch { }
try { $OutputEncoding = [System.Text.Encoding]::UTF8 } catch { }

$SiteDir = Split-Path -Parent $MyInvocation.MyCommand.Path
$Url = 'http://127.0.0.1:' + $Port + '/'

Write-Host ''
Write-Host '  سامانهٔ جامع منابع انسانی — اچ‌آر اپ' -ForegroundColor Cyan
Write-Host '  سایت معرفی و دموی تبلیغاتی' -ForegroundColor DarkGray
Write-Host ''
Write-Host ('  نشانی سایت  : ' + $Url) -ForegroundColor White
Write-Host ('  پوشهٔ سایت    : ' + $SiteDir) -ForegroundColor DarkGray
Write-Host '  پایان کار    : Ctrl+C یا بستن همین پنجره' -ForegroundColor DarkGray
Write-Host ''

# --- پیدا کردن پایتون (python یا py) -------------------------------------
$py = Get-Command python -ErrorAction SilentlyContinue
if (-not $py) { $py = Get-Command py -ErrorAction SilentlyContinue }

if (-not $py) {
  Write-Host '  پایتون روی این رایانه پیدا نشد؛ سایت مستقیم در مرورگر باز می‌شود.' -ForegroundColor Yellow
  if (-not $NoOpen) { Start-Process (Join-Path $SiteDir 'index.html') }
  return
}

# --- باز کردن مرورگر کمی بعد از بالا آمدن سرور --------------------------
$opener = $null
if (-not $NoOpen) {
  $opener = Start-Job -ScriptBlock {
    param($u)
    Start-Sleep -Seconds 2
    Start-Process $u
  } -ArgumentList $Url
}

# --- اجرای سرور در همین پنجره (با Ctrl+C بسته می‌شود) -------------------
try {
  & $py.Source -m http.server $Port --bind 127.0.0.1 --directory $SiteDir
}
catch {
  Write-Host ''
  Write-Host ('  سرور بالا نیامد: ' + $_.Exception.Message) -ForegroundColor Red
  Write-Host ('  اگر درگاه ' + $Port + ' اشغال است، همین اسکریپت را با درگاه دیگری اجرا کنید:') -ForegroundColor DarkGray
  Write-Host ('    powershell -NoProfile -ExecutionPolicy Bypass -File "' + $MyInvocation.MyCommand.Path + '" -Port 8097') -ForegroundColor DarkGray
}
finally {
  if ($opener) { Remove-Job -Job $opener -Force -ErrorAction SilentlyContinue }
  Write-Host ''
  Write-Host '  سرور سایت بسته شد. روز خوش!' -ForegroundColor Cyan
  Write-Host ''
}
