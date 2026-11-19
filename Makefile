# دستورهای توسعهٔ سامانه.
#
# «make check» همان چیزی است که CI اجرا می‌کند: انحراف قرارداد، مرز ماژول، تطبیق
# قواعد و آزمون‌های سرور و هسته. پیش از هر کامیت همین را اجرا کنید.

.PHONY: help gen gen-demo-schema check verify test-go test-core conformance smoke-postgres run-server run-bonusd bonusd bonusd-386 bonusapp bonusapp-386 run-bonusapp fmt mobile-setup mobile-test android-libs pentest skills-check

# مسیرهای بستهٔ بازبینی امنیتی (پنتست)؛ در خط فرمان قابل تغییرند:
#   make pentest PENTEST_OUT=D:/hrapptest PENTEST_ZIP=D:/hrapptest.zip
PENTEST_OUT ?= dist-pentest/hrms-pentest-src
PENTEST_ZIP ?= dist-pentest/hrapp-pentest.zip

help:
	@echo "دستورهای موجود:"
	@echo "  make gen           بازتولید خروجی‌ها از مانیفست ماژول‌ها و قرارداد"
	@echo "  make gen-demo-schema  نمای ساختار صفحههای زنده در نمونهٔ وب (نیازمند سرور بالا)"
	@echo "  make verify        دروازه‌های معماری، مرز ماژول و انحراف قرارداد"
	@echo "  make test-core     آزمون‌های هستهٔ Rust"
	@echo "  make test-go       آزمون‌های سرور Go (شامل تطبیق قواعد)"
	@echo "  make conformance   دروازهٔ تطبیق قواعد هسته با fixtureهای طلایی"
	@echo "  make smoke-postgres آزمون دودی روی Postgres (نیازمند HRMS_TEST_DATABASE_URL)"
	@echo "  make run-server    اجرای سرور + رابط وب روی http://127.0.0.1:8080/"
	@echo "  make bonusd        ساخت باینری «سیستم پاداش» (سرور/bin/bonusd)"
	@echo "  make bonusd-386    ساخت همان باینری برای ویندوز ۳۲بیتی (x86)"
	@echo "  make run-bonusd    اجرای سیستم پاداش روی http://127.0.0.1:8099/"
	@echo "  make bonusapp      ساخت اپ دسکتاپ ویندوزی ۶۴بیتی (پنجرهٔ بومی WebView2)"
	@echo "  make bonusapp-386  ساخت همان اپ برای ویندوز ۳۲بیتی (x86)"
	@echo "  make run-bonusapp  اجرای اپ دسکتاپ روی همین ویندوز"
	@echo "  make mobile-test   آزمون‌های کلاینت React Native"
	@echo "  make android-libs  ساخت کتابخانهٔ بومی هسته برای اندروید"
	@echo "  make pentest       بستهٔ تست نفوذ: رونویسی تمیز + زیپ استاندارد + چک‌سام"
	@echo "  make skills-check  اعتبارسنجی کتابخانهٔ اسکیل‌های عامل (.claude/skills)"

gen:
	node tools/gen/gen.mjs

# نمای ساختار صفحههای زنده برای نمونهٔ وبِ بدون سرور: از /schema خودِ سرور خوانده میشود،
# پس سرور محلی باید بالا باشد (start-HRMS.bat یا make run-server).
#   make gen-demo-schema           ساخت/بهروزرسانی فایل تولیدشده
#   make gen-demo-schema CHECK=--check   فقط بررسی همخوانی با سرور
gen-demo-schema:
	node tools/gen/demo-schema.mjs $(CHECK)

verify:
	node tools/gen/gen.mjs --check
	node tools/check/verify.mjs
	node tools/skills/check-skills.mjs
	node tools/check/security-spec.mjs
	node tools/check/frontend-security.mjs
	node tools/check/ops-spec.mjs

# اعتبارسنجی قالب اسکیل‌های پروژه بر پایهٔ استاندارد Agent Skills (name/description/ساختار).
skills-check:
	node tools/skills/check-skills.mjs

check: verify test-go test-core conformance

test-core:
	cd core && cargo test --workspace --all-features

conformance:
	cd core && cargo run -p hrms-conformance

test-go:
	cd server && go test -race -count=1 ./...

# آزمون دودی روی پایگاه‌دادهٔ واقعی: سرور با Postgres بالا می‌آید (مهاجرت + بذرکاری نمونه)
# و سپس پُر بودن همهٔ ماژول‌ها، چرخهٔ ایجاد/جستجو/ویرایش/حذف روی هر نوع رکورد و خواندن
# رکورد از دو نمونهٔ سرور روی همان پایگاه‌داده سنجیده می‌شود. در CI روی سرویس Postgres
# اجرا می‌شود؛ محلی باید خودتان DSN یک پایگاه‌دادهٔ آزمایشی را بدهید (دادهٔ نمونه در آن
# ساخته و رکوردهای آزمون پاک می‌شوند):
#   HRMS_TEST_DATABASE_URL="postgres://hrms@127.0.0.1:5432/hrms_smoke?sslmode=disable" make smoke-postgres
smoke-postgres:
	@test -n "$$HRMS_TEST_DATABASE_URL" || (echo "✖ متغیر HRMS_TEST_DATABASE_URL تنظیم نشده است (پایگاه‌دادهٔ آزمایشی)"; exit 1)
	cd server && go test -count=1 -v -run PostgresSmoke ./internal/bootstrap/

run-server:
	cd server && infisical run --env=dev -- go run ./cmd/hrmsd -addr 127.0.0.1:8080 -web ../design/prototype -env .env

# «سیستم پاداش»: تنها ماژول پاداش، بدون پایگاه‌داده، با رابط وب جاسازی‌شده.
# خروجی یک فایل اجرایی است که روی هر رایانه‌ای با همان سیستم‌عامل اجرا می‌شود؛
# فقط پوشهٔ server/bonus-data را کنارش نگه دارید (پشتیبان‌گیری = کپی همان پوشه).
bonusd:
	cd server && go build -o bin/bonusd ./cmd/bonusd

# نسخهٔ ۳۲بیتی برای ویندوزهای x86. به‌جز معماری، هیچ تفاوتی با نسخهٔ ۶۴بیتی ندارد:
# ساخت همچنان بدون CGO است، پس باز هم فقط یک فایل اجرایی تحویل می‌شود.
bonusd-386:
	cd server && GOOS=windows GOARCH=386 CGO_ENABLED=0 go build -trimpath \
		-ldflags "-s -w" -o bin/bonusd-386.exe ./cmd/bonusd

run-bonusd: bonusd
	cd server && ./bin/bonusd -addr 127.0.0.1:8099 -open

# «سیستم پاداش» به شکل اپ دسکتاپ ویندوزی: پنجرهٔ بومی WebView2، بدون کنسول،
# بدون نوار نشانی. سرور داخل خود برنامه اجرا می‌شود و داده کنار فایل اجرایی می‌نشیند.
# ساخت بدون CGO است، پس موتور WebView2 داخل فایل اجرایی جاسازی می‌شود و هیچ DLL
# یا زنجیرهٔ ابزار دیگری همراه بسته نمی‌رود.
bonusapp:
	cd server && GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -trimpath \
		-ldflags "-s -w -H=windowsgui" -o bin/bonusapp.exe ./cmd/bonusapp

# نسخهٔ ۳۲بیتی اپ دسکتاپ برای ویندوز x86. موتور WebView2 با معماری سامانه یکسان
# می‌شود، پس روی ویندوز ۳۲بیتی که موتور x86 دارد، پنجرهٔ بومی همان‌گونه باز می‌شود؛
# روی سامانه‌ای که فقط موتور x64 نصب دارد، برنامه به‌جای خطا مسیر جایگزین مرورگر را
# نشان می‌دهد (بخش محدودیت‌ها در docs/bonus-standalone-fa.md).
bonusapp-386:
	cd server && GOOS=windows GOARCH=386 CGO_ENABLED=0 go build -trimpath \
		-ldflags "-s -w -H=windowsgui" -o bin/bonusapp-386.exe ./cmd/bonusapp

run-bonusapp: bonusapp
	cd server && ./bin/bonusapp.exe

fmt:
	cd core && cargo fmt --all
	cd server && gofmt -w .

mobile-setup:
	cd apps/mobile && npm install

mobile-test:
	cd apps/mobile && npm run typecheck && npm test

android-libs:
	cd core && cargo ndk -t arm64-v8a -t x86_64 -o ../apps/mobile/android/app/src/main/jniLibs build -p hrms-ffi --release

# بستهٔ بازبینی امنیتی (تست نفوذ) برای تحویل به ابزارها و مدل‌های دیگر.
#
# سه گام: رونویسی تمیز مخزن (بدون اسرار، لاگ، باینری و دادهٔ زمان اجرا) → زیپ استاندارد
# با جداکنندهٔ «/» و پرچم UTF-8 (DEFLATE جایی که سود دارد) → چک‌سام SHA-256 کنار زیپ.
# خروجی پیش‌فرض داخل مخزن و در gitignore است؛ برای مسیر دیگر با PENTEST_OUT/PENTEST_ZIP بدهید.
pentest:
	node tools/pentest/build-pentest.mjs --out $(PENTEST_OUT)
	node tools/pentest/zip-folder.mjs $(PENTEST_OUT) $(PENTEST_ZIP) --deflate
	node -e "const c=require('crypto'),f=require('fs'),p=require('path');const z=process.argv[1];f.writeFileSync(z+'.sha256',c.createHash('sha256').update(f.readFileSync(z)).digest('hex')+'  '+p.basename(z)+'\n');console.log('sha256: '+p.basename(z)+'.sha256')" "$(PENTEST_ZIP)"
