# راه‌اندازی n8n برای یکپارچه‌سازی با سامانهٔ منابع انسانی.
#
#   powershell -NoProfile -ExecutionPolicy Bypass -File tools\n8n\n8n.ps1            # اجرا (پنجرهٔ همین فرمان)
#   powershell -NoProfile -ExecutionPolicy Bypass -File tools\n8n\n8n.ps1 -Start   # اجرا در پس‌زمینه با لاگ
#   powershell -NoProfile -ExecutionPolicy Bypass -File tools\n8n\n8n.ps1 -Status
#   powershell -NoProfile -ExecutionPolicy Bypass -File tools\n8n\n8n.ps1 -Stop
#
# چرا Node 22 جداگانه: n8n با نسخه‌های ۲۰/۲۲/۲۴ آزموده شده؛ این ماشین Node 26 دارد و
# نصب هم‌زیست جلوی شلوغ‌شدن نسخهٔ سیستم را می‌گیرد. مسیر هر دو با متغیر قابل تغییر است.
#
# دادهٔ n8n (جریان‌ها، کاربران، رمزهای ذخیره‌شده) در پوشهٔ -DataDir می‌ماند؛ این پوشه با
# N8N_ENCRYPTION_KEY رمز می‌شود، پس کلید باید کنارش نگه داشته و در پشتیبان‌گیری حفظ شود.

param(
  [switch]$Start,
  [switch]$Stop,
  [switch]$Status,
  [string]$DataDir = (Join-Path $env:LOCALAPPDATA 'tools\n8n-data'),
  [int]$Port = 5678
)

$ErrorActionPreference = 'Stop'
$repo = Split-Path -Parent (Split-Path -Parent $PSScriptRoot)
if (-not (Test-Path (Join-Path $repo 'server'))) { $repo = 'D:\hrapp' }
$nodeDir = Join-Path $env:LOCALAPPDATA 'tools\node22'
$prefix = Join-Path $env:LOCALAPPDATA 'tools\n8n'
$n8nExe = Join-Path $prefix 'n8n.cmd'
$pidFile = Join-Path $DataDir 'n8n.pid'
$logDir = Join-Path $DataDir 'logs'
$logFile = Join-Path $logDir 'n8n.log'
$envFile = Join-Path $repo 'integrations\n8n\.env'
$keyFile = Join-Path $DataDir 'encryption.key'

function Get-Running {
  if (-not (Test-Path $pidFile)) { return $null }
  $p = Get-Content $pidFile -ErrorAction SilentlyContinue
  if (-not $p) { return $null }
  $proc = Get-Process -Id ([int]$p) -ErrorAction SilentlyContinue
  if ($null -eq $proc) { return $null }
  return $proc
}

if ($Status) {
  $proc = Get-Running
  if ($proc) {
    "n8n در حال اجرا است — PID $($proc.Id)"
    try {
      $r = Invoke-WebRequest -Uri "http://127.0.0.1:$Port/healthz" -UseBasicParsing -TimeoutSec 5
      "پاسخ /healthz: $($r.StatusCode)"
    } catch {
      "‏/healthz پاسخ نداد: $($_.Exception.Message)"
    }
  } else {
    'n8n اجرا نمی‌شود.'
  }
  "پوشهٔ داده: $DataDir"
  "لاگ: $logFile"
  exit 0
}

if ($Stop) {
  $proc = Get-Running
  if (-not $proc) { 'n8n اجرا نمی‌شود.'; exit 0 }
  Stop-Process -Id $proc.Id -Force
  Remove-Item $pidFile -ErrorAction SilentlyContinue
  "n8n متوقف شد (PID $($proc.Id))."
  exit 0
}

if (-not (Test-Path $n8nExe)) {
  throw "n8n نصب نیست ($n8nExe). با Node 22: `"$nodeDir\npm.cmd`" install -g n8n --prefix `"$prefix`""
}
if (-not (Test-Path (Join-Path $nodeDir 'node.exe'))) {
  throw "Node 22 نصب نیست ($nodeDir)."
}

New-Item -ItemType Directory -Force -Path $DataDir, $logDir | Out-Null

# کلید رمز دادهٔ n8n: یک‌بار ساخته و نگه داشته می‌شود؛ عوض شدنش رمزهای ذخیره‌شده را
# ناخوانا می‌کند.
if (-not (Test-Path $keyFile)) {
  $bytes = New-Object byte[] 32
  [Security.Cryptography.RandomNumberGenerator]::Create().GetBytes($bytes)
  [System.IO.File]::WriteAllText($keyFile, [Convert]::ToBase64String($bytes), [Text.UTF8Encoding]::new($false))
  "کلید رمز تازه ساخته شد: $keyFile"
}
$env:N8N_ENCRYPTION_KEY = (Get-Content $keyFile -Raw).Trim()

$env:PATH = "$nodeDir;$prefix;$env:PATH"
$env:N8N_USER_FOLDER = $DataDir
$env:N8N_HOST = '127.0.0.1'
$env:N8N_PORT = "$Port"
$env:N8N_PROTOCOL = 'http'
$env:N8N_DIAGNOSTICS_ENABLED = 'false'
$env:N8N_VERSION_NOTIFICATIONS_ENABLED = 'false'
$env:N8N_TEMPLATES_ENABLED = 'false'
$env:N8N_PERSONALIZATION_ENABLED = 'false'
$env:N8N_LOG_LEVEL = 'info'
# نود Code برای بازبینی امضای HMAC به crypto نیاز دارد.
$env:NODE_FUNCTION_ALLOW_BUILTIN = 'crypto'
# n8n نسخهٔ ۲ دسترسی نودهای Code/عبارت‌ها به متغیرهای محیطی را پیش‌فرض می‌بندد و گردش‌کار
# با خطای «access to env vars denied» شکست می‌خورد؛ در این نصب، کلید وبهوک و نشانی API
# از همین متغیرهای سطح فرایند می‌آیند، پس دسترسی باز می‌شود (و n8n فقط روی 127.0.0.1
# شنود می‌کند تا سطح تماس محدود بماند).
$env:N8N_BLOCK_ENV_ACCESS_IN_NODE = 'false'
$env:N8N_RUNNERS_ENABLED = 'true'
$env:GENERIC_TIMEZONE = 'Asia/Tehran'
$env:TZ = 'Asia/Tehran'

# متغیرهای خودِ یکپارچه‌سازی (کلید وبهوک، نشانی API و توکن) از فایل .env خوانده می‌شوند
# تا در مخزن نباشند.
if (Test-Path $envFile) {
  foreach ($line in Get-Content $envFile -Encoding UTF8) {
    $t = $line.Trim()
    if ($t -eq '' -or $t.StartsWith('#')) { continue }
    $eq = $t.IndexOf('=')
    if ($eq -lt 1) { continue }
    $k = $t.Substring(0, $eq).Trim()
    $v = $t.Substring($eq + 1).Trim().Trim('"').Trim("'")
    Set-Item -Path "env:$k" -Value $v
  }
} else {
  "هشدار: فایل $envFile نیست؛ متغیرهای HRMS_* از محیط خوانده می‌شوند."
}

if ($Start) {
  $existing = Get-Running
  if ($existing) { "n8n از قبل اجرا می‌شود (PID $($existing.Id))."; exit 0 }
  # اجرا با node.exe و فایل bin خودِ n8n، نه با شیم n8n.cmd: شیم یک فرایند cmd در میان
  # می‌گذارد و PIDی که ذخیره می‌شود مال همان cmd است، نه فرایند واقعی n8n؛ آن‌وقت -Stop
  # نمی‌تواند n8n را ببندد و پورت اشغال می‌ماند.
  $n8nBin = Join-Path $prefix 'node_modules\n8n\bin\n8n'
  $nodeExe = Join-Path $nodeDir 'node.exe'
  if ((Test-Path $n8nBin) -and (Test-Path $nodeExe)) {
    $proc = Start-Process -FilePath $nodeExe -ArgumentList "`"$n8nBin`"", 'start' -WorkingDirectory $DataDir `
      -RedirectStandardOutput $logFile -RedirectStandardError "$logFile.err" `
      -WindowStyle Hidden -PassThru
  } else {
    $proc = Start-Process -FilePath $n8nExe -ArgumentList 'start' -WorkingDirectory $DataDir `
      -RedirectStandardOutput $logFile -RedirectStandardError "$logFile.err" `
      -WindowStyle Hidden -PassThru
  }
  Set-Content -Path $pidFile -Value $proc.Id
  "n8n در پس‌زمینه شروع شد (PID $($proc.Id)). رابط: http://127.0.0.1:$Port/"
  "لاگ: $logFile"
  exit 0
}

"n8n: http://127.0.0.1:$Port/  (متوقف‌کردن: Ctrl+C، سرویس پس‌زمینه: -Stop)"
& $n8nExe start