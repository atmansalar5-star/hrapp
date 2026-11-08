# make-pdf.ps1 - build the printable PDFs of the site with headless Chrome.
# Usage:  .\make-pdf.cmd            (builds both PDFs)
#         powershell -File .\make-pdf.ps1 -Only presentation
#
# Notes (important, kept ASCII on purpose):
#  * Chrome/Edge headless prints the page with print media, so site.css @media print applies.
#  * The PDF engine cannot embed *variable* font files, so the print layer declares the
#    static Vazirmatn faces (VazirPrint). Without them Persian text falls back to Tahoma.
#  * --allow-file-access-from-files lets the local woff2 fonts load from file:// URLs.

param(
  [ValidateSet('all', 'presentation', 'maroun', 'demo')]
  [string]$Only = 'all'
)

$ErrorActionPreference = 'Stop'
$here = Split-Path -Parent $MyInvocation.MyCommand.Path

$browsers = @(
  'C:\Program Files\Google\Chrome\Application\chrome.exe',
  'C:\Program Files (x86)\Google\Chrome\Application\chrome.exe',
  'C:\Program Files (x86)\Microsoft\Edge\Application\msedge.exe',
  'C:\Program Files\Microsoft\Edge\Application\msedge.exe'
)
$chrome = $browsers | Where-Object { Test-Path $_ } | Select-Object -First 1
if (-not $chrome) { throw 'Chrome or Edge not found. Install one of them first.' }
Write-Host "browser: $chrome"

$jobs = @(
  @{ html = 'presentation.html';        pdf = 'hrapp-presentation.pdf' },
  @{ html = 'presentation-maroun.html'; pdf = 'hrapp-presentation-maroun.pdf' },
  @{ html = 'demo.html';                pdf = 'hrapp-demo.pdf' }
)
if ($Only -ne 'all') { $jobs = $jobs | Where-Object { $_.html -like "$Only.html" } }

$profile = Join-Path $env:TEMP ('chrome-pdf-' + [guid]::NewGuid().ToString('N').Substring(0, 8))

foreach ($j in $jobs) {
  $src = Join-Path $here $j.html
  $dst = Join-Path $here $j.pdf
  if (-not (Test-Path $src)) { throw "missing source: $src" }
  if (Test-Path $dst) { Remove-Item $dst -Force }

  $uri = ([uri]$src).AbsoluteUri
  $args = @(
    '--headless=new',
    '--disable-gpu',
    '--no-first-run',
    '--no-pdf-header-footer',
    '--allow-file-access-from-files',
    '--virtual-time-budget=25000',
    "--user-data-dir=$profile",
    "--print-to-pdf=$dst",
    $uri
  )
  $sw = [System.Diagnostics.Stopwatch]::StartNew()
  $proc = Start-Process -FilePath $chrome -ArgumentList $args -PassThru -Wait -WindowStyle Hidden
  $sw.Stop()

  if (-not (Test-Path $dst)) { throw "PDF was not written: $dst" }
  $kb = [math]::Round((Get-Item $dst).Length / 1KB)
  $txt = [System.Text.Encoding]::GetEncoding(28591).GetString([System.IO.File]::ReadAllBytes($dst))
  $pages = ([regex]::Matches($txt, '/Type\s*/Page[\s/>]')).Count
  $imgs = ([regex]::Matches($txt, '/Subtype\s*/Image')).Count
  $vazir = [bool]([regex]::IsMatch($txt, '/BaseFont\s*/[A-Z]+\+Vazirmatn'))
  $ok = if ($vazir) { 'Vazirmatn OK' } else { 'WARNING: font fallback!' }
  Write-Host ("{0} -> {1}  pages={2} images={3} size={4}KB  {5}  ({6}s)" -f $j.html, $j.pdf, $pages, $imgs, $kb, $ok, [math]::Round($sw.Elapsed.TotalSeconds))
}

Remove-Item $profile -Recurse -Force -ErrorAction SilentlyContinue
Write-Host 'done.'