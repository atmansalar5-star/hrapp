# =====================================================================
#  ساخت میان‌بر (Shortcut) سایت معرفی «سامانهٔ جامع منابع انسانی»
# ---------------------------------------------------------------------
#  این اسکریپت یک میان‌بر روی «دسکتاپ» و یکی در «منوی شروع» می‌سازد که
#  با یک کلیک، سرور محلی سایت را بالا می‌آورد و سایت را در مرورگر باز
#  می‌کند. اگر پوشهٔ سایت را جابه‌جا کردید، همین اسکریپت را دوباره اجرا
#  کنید تا میان‌برها با مسیر تازه ساخته شوند.
#
#  چرا دو مرحله‌ای: میان‌بر نخست با یک نام لاتین ساخته و سپس با
#  System.IO.File به نام فارسی تغییرنام می‌دهد. علتش این است که واسط
#  COM ویندوز (WScript.Shell) مسیر غیرلاتین را درست نمی‌پذیرد و خطای
#  «Unable to save shortcut» می‌دهد؛ خودِ فایل .lnk اما نام فارسی را
#  بی هیچ مشکلی نگه می‌دارد.
#
#  اجرا:  powershell -NoProfile -ExecutionPolicy Bypass -File build-shortcut.ps1
#  گزینه‌ها: -Name "نام دلخواه.lnk"  -NoDesktop  -NoStartMenu
# =====================================================================
param(
  [string]$Name = 'سایت معرفی سامانه اچ آر اپ.lnk',
  [switch]$NoDesktop,
  [switch]$NoStartMenu
)

$ErrorActionPreference = 'Stop'
try { [Console]::OutputEncoding = [System.Text.Encoding]::UTF8 } catch { }

$SiteDir = Split-Path -Parent $MyInvocation.MyCommand.Path
$Target  = Join-Path $SiteDir 'start-site.cmd'
$Icon    = Join-Path $SiteDir 'assets\icon.ico'
$Temp    = '_hr-app-site-temp.lnk'

if (-not (Test-Path -LiteralPath $Target)) {
  Write-Host ('  فایل راه‌انداز پیدا نشد: ' + $Target) -ForegroundColor Red
  exit 1
}

$places = @()
if (-not $NoDesktop)   { $places += [Environment]::GetFolderPath('Desktop') }
if (-not $NoStartMenu) { $places += (Join-Path $env:APPDATA 'Microsoft\Windows\Start Menu\Programs') }

$shell = New-Object -ComObject WScript.Shell
$done = 0

foreach ($dir in $places) {
  if (-not $dir -or -not (Test-Path -LiteralPath $dir)) { continue }
  $final   = Join-Path $dir $Name
  $scratch = Join-Path $dir $Temp

  if (Test-Path -LiteralPath $scratch) { Remove-Item -LiteralPath $scratch -Force }
  $link = $shell.CreateShortcut($scratch)
  $link.TargetPath       = $Target
  $link.WorkingDirectory = $SiteDir
  $link.Description      = 'سایت معرفی و دموی تبلیغاتی سامانه جامع منابع انسانی'
  $link.WindowStyle      = 1
  if (Test-Path -LiteralPath $Icon) { $link.IconLocation = $Icon + ',0' }
  $link.Save()

  if (Test-Path -LiteralPath $final) { Remove-Item -LiteralPath $final -Force }
  [System.IO.File]::Move($scratch, $final)

  if (Test-Path -LiteralPath $final) {
    $size = (Get-Item -LiteralPath $final).Length
    Write-Host ('  میان‌بر ساخته شد: ' + $final + '  (' + $size + ' بایت)') -ForegroundColor Green
    $done++
  }
}

if ($done -eq 0) {
  Write-Host '  هیچ میان‌بری ساخته نشد (پوشهٔ مقصد پیدا نشد).' -ForegroundColor Yellow
  exit 1
}

Write-Host ''
Write-Host '  با یک کلیک روی میان‌بر، سرور محلی بالا می‌آید و سایت باز می‌شود:' -ForegroundColor Cyan
Write-Host '  http://127.0.0.1:8098/' -ForegroundColor White
Write-Host '  برای بستن سرور، پنجرهٔ کنسول را ببندید یا Ctrl+C بزنید.' -ForegroundColor DarkGray
Write-Host ''
