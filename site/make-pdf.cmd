@echo off
rem Build the printable PDFs (hrapp-presentation.pdf and hrapp-demo.pdf)
rem from presentation.html and demo.html using headless Chrome/Edge.
powershell -NoProfile -ExecutionPolicy Bypass -File "%~dp0make-pdf.ps1" %*
if errorlevel 1 pause