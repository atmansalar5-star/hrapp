// آزمون جریان سرتاسری «نشان شرکت، QR کد و تأیید سند»: بارگذاری لوگو از API، سرو عمومی
// فایل، QR کد شرکت/کارمند/سند، صفحهٔ تأیید عمومی و امضای نشانی سند چاپی.

package bootstrap

import (
	"bytes"
	"encoding/json"
	"image"
	"image/color"
	"image/png"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"

	"hrms/internal/contracts"
	"hrms/internal/platform/qr"
)

// testKey کلید امضای نشانی تأیید در آزمون (همان JWTSecret سرور آزمون).
var testKey = []byte("test-secret-key-for-integration")

// multipartPNG بدنهٔ multipart بارگذاری یک PNG کوچک می‌سازد.
func multipartPNG(t *testing.T) ([]byte, string) {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 4, 4))
	img.Set(1, 1, color.RGBA{R: 30, G: 136, B: 229, A: 255})
	var imageBuf bytes.Buffer
	if err := png.Encode(&imageBuf, img); err != nil {
		t.Fatalf("ساخت تصویر آزمون ناموفق: %v", err)
	}
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("file", "logo.png")
	if err != nil {
		t.Fatalf("ساخت بخش فایل ناموفق: %v", err)
	}
	if _, err := part.Write(imageBuf.Bytes()); err != nil {
		t.Fatalf("نوشتن تصویر ناموفق: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("بستن بدنهٔ multipart ناموفق: %v", err)
	}
	return body.Bytes(), writer.FormDataContentType()
}

func TestBrandingAndQRFlow(t *testing.T) {
	h := newTestServer(t)
	admin := login(t, h, "admin", "demo").AccessToken

	// ---------- شرکت فعال را پیدا می‌کنیم (نشان روی همین شرکت بارگذاری می‌شود) ----------
	list := call(t, h, http.MethodGet, "/api/v1/settings/records?kind=companyProfiles&size=100", admin, nil)
	if list.Code != http.StatusOK {
		t.Fatalf("فهرست شرکت‌ها باید ۲۰۰ بدهد، %d داد", list.Code)
	}
	var page contracts.RecordPage
	decodeInto(t, list, &page)
	var active contracts.Record
	for _, item := range page.Items {
		if item.Fields["isActiveDeployment"] == "بله" {
			active = item
		}
	}
	if active.ID == "" {
		t.Fatal("شرکت فعال در فهرست شرکت‌ها نبود")
	}
	code := active.Fields["code"]

	// ---------- درخواست بدون توکن رد می‌شود ----------
	if got := call(t, h, http.MethodPost, "/api/v1/settings/companies/"+active.ID+"/logo", "", nil); got.Code != http.StatusUnauthorized {
		t.Errorf("بارگذاری نشان بدون توکن باید ۴۰۱ بدهد، %d داد", got.Code)
	}

	// ---------- بارگذاری نشان ----------
	body, contentType := multipartPNG(t)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/settings/companies/"+active.ID+"/logo", bytes.NewReader(body))
	req.Header.Set("Content-Type", contentType)
	req.Header.Set("Authorization", "Bearer "+admin)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("بارگذاری نشان باید ۲۰۰ بدهد، %d داد: %s", rec.Code, rec.Body.String())
	}
	var uploaded struct {
		Company contracts.Record `json:"company"`
		Logo    struct {
			Name, URL, ContentType, SHA256 string
			SizeBytes                      int
		} `json:"logo"`
		NoteFa string `json:"noteFa"`
	}
	decodeInto(t, rec, &uploaded)
	if uploaded.Logo.URL == "" || !strings.HasPrefix(uploaded.Logo.URL, "/api/v1/branding/") {
		t.Fatalf("نشانی نشان درست نیست: %+v", uploaded.Logo)
	}
	if uploaded.Company.Fields["logoFile"] != uploaded.Logo.URL {
		t.Errorf("نشانی نشان باید روی رکورد شرکت بنشیند: %+v", uploaded.Company.Fields)
	}
	if uploaded.Company.Fields["logoHash"] == "" || uploaded.Company.Fields["logoUpdatedAt"] == "" {
		t.Errorf("اثر انگشت و تاریخ بارگذاری نشان باید ثبت شود: %+v", uploaded.Company.Fields)
	}
	// بقیهٔ اطلاعات شرکت نباید در ویرایش نشان از دست برود.
	if uploaded.Company.Fields["holdingFa"] != active.Fields["holdingFa"] {
		t.Errorf("هلدینگ شرکت پس از بارگذاری نشان تغییر کرده: %q", uploaded.Company.Fields["holdingFa"])
	}
	if uploaded.NoteFa == "" {
		t.Errorf("پاسخ بارگذاری باید یادداشت فارسی داشته باشد")
	}

	// ---------- سرو عمومی فایل نشان (بدون توکن، مثل صفحهٔ ورود) ----------
	fileURL := uploaded.Logo.URL
	got := call(t, h, http.MethodGet, fileURL, "", nil)
	if got.Code != http.StatusOK {
		t.Fatalf("سرو فایل نشان باید ۲۰۰ بدهد، %d داد", got.Code)
	}
	if ct := got.Header().Get("Content-Type"); !strings.Contains(ct, "image/png") {
		t.Errorf("نوع محتوای نشان باید PNG باشد، «%s» شد", ct)
	}
	if got.Body.Len() == 0 {
		t.Errorf("بدنهٔ فایل نشان خالی است")
	}
	if missing := call(t, h, http.MethodGet, "/api/v1/branding/nope-000.png", "", nil); missing.Code != http.StatusNotFound {
		t.Errorf("فایل نبوده باید ۴۰۴ بدهد، %d داد", missing.Code)
	}

	// ---------- نشان شرکت فعال ----------
	activeLogo := call(t, h, http.MethodGet, "/api/v1/branding/active", "", nil)
	if activeLogo.Code != http.StatusOK {
		t.Fatalf("نشان شرکت فعال باید ۲۰۰ بدهد، %d داد", activeLogo.Code)
	}
	var activeBody struct {
		Org             string `json:"org"`
		LogoURL         string `json:"logoUrl"`
		HasUploadedLogo bool   `json:"hasUploadedLogo"`
	}
	decodeInto(t, activeLogo, &activeBody)
	if activeBody.LogoURL != fileURL || !activeBody.HasUploadedLogo {
		t.Errorf("نشان شرکت فعال باید همان فایل بارگذاری‌شده باشد: %+v", activeBody)
	}

	// ---------- QR کد شرکت: عمومی است چون محتوایش فقط نشانی صفحهٔ تأیید شرکت است ----------
	svg := call(t, h, http.MethodGet, "/api/v1/qr/company/"+code, "", nil)
	if svg.Code != http.StatusOK || !strings.Contains(svg.Body.String(), "<svg") {
		t.Fatalf("QR کد شرکت باید SVG بدهد (%d): %.80s", svg.Code, svg.Body.String())
	}
	if ct := svg.Header().Get("Content-Type"); !strings.Contains(ct, "image/svg+xml") {
		t.Errorf("نوع محتوای QR باید SVG باشد، «%s» شد", ct)
	}
	// QR کارمند شناسه‌دار است و بی‌توکن ساخته نمی‌شود (برچسب img هم برای آن به کار نمی‌رود).
	if noAuth := call(t, h, http.MethodGet, "/api/v1/qr/employee/EMP-001", "", nil); noAuth.Code != http.StatusUnauthorized {
		t.Errorf("ساخت QR کارمند بدون توکن باید ۴۰۱ بدهد، %d داد", noAuth.Code)
	}
	link := call(t, h, http.MethodGet, "/api/v1/qr/company/"+code+"?format=json", admin, nil)
	var linkBody struct {
		Link string `json:"link"`
	}
	decodeInto(t, link, &linkBody)
	if !strings.Contains(linkBody.Link, "/verify/company/"+code) {
		t.Errorf("نشانی QR شرکت باید به صفحهٔ تأیید اشاره کند: %s", linkBody.Link)
	}
	pngQR := call(t, h, http.MethodGet, "/api/v1/qr/employee/EMP-001?format=png&size=160", admin, nil)
	if pngQR.Code != http.StatusOK || pngQR.Body.Len() < 100 {
		t.Errorf("QR کد کارمند به شکل PNG ساخته نشد (%d)", pngQR.Code)
	}

	// ---------- صفحهٔ تأیید عمومی شرکت (بدون توکن) ----------
	pageHTML := call(t, h, http.MethodGet, "/verify/company/"+code, "", nil)
	if pageHTML.Code != http.StatusOK {
		t.Fatalf("صفحهٔ تأیید شرکت باید ۲۰۰ بدهد، %d داد", pageHTML.Code)
	}
	if !strings.Contains(pageHTML.Body.String(), active.Title) {
		t.Errorf("صفحهٔ تأیید باید نام شرکت را نشان دهد: %.200s", pageHTML.Body.String())
	}
	unknown := call(t, h, http.MethodGet, "/verify/company/nope_co", "", nil)
	if unknown.Code != http.StatusNotFound {
		t.Errorf("شرکت ناشناخته باید ۴۰۴ بدهد، %d داد", unknown.Code)
	}

	// ---------- تأیید کارمند ----------
	verify := call(t, h, http.MethodGet, "/api/v1/verify/employee/EMP-001", "", nil)
	if verify.Code != http.StatusOK {
		t.Fatalf("تأیید کارمند باید ۲۰۰ بدهد، %d داد", verify.Code)
	}
	var verifyBody map[string]any
	decodeInto(t, verify, &verifyBody)
	if verifyBody["verified"] != true || verifyBody["fullName"] == nil {
		t.Errorf("پاسخ تأیید کارمند باید نام و تأیید داشته باشد: %+v", verifyBody)
	}
	if _, hasSalary := verifyBody["baseSalary"]; hasSalary {
		t.Errorf("صفحهٔ تأیید نباید مبلغ یا دادهٔ پرسنلی حساس بدهد: %+v", verifyBody)
	}

	// ---------- سند چاپی: QR تأیید و امضای نشانی ----------
	doc := call(t, h, http.MethodGet, "/api/v1/documents/employment_certificate/EMP-001", admin, nil)
	if doc.Code != http.StatusOK {
		t.Fatalf("چاپ گواهی اشتغال باید ۲۰۰ بدهد، %d داد: %s", doc.Code, doc.Body.String())
	}
	docBody := doc.Body.String()
	if !strings.Contains(docBody, "<svg") || !strings.Contains(docBody, "برای بررسی اصالت سند") {
		t.Errorf("سند چاپی باید QR تأیید داشته باشد")
	}
	serial := ""
	if m := regexp.MustCompile(`شمارهٔ سند:\s*([^<\s]+)`).FindStringSubmatch(docBody); len(m) > 1 {
		serial = strings.TrimSpace(m[1])
	}
	if serial == "" {
		t.Fatalf("سریال سند از متن چاپی خوانده نشد")
	}
	token := qr.Sign(testKey, "document", "employment_certificate", "EMP-001", serial)
	okVerify := call(t, h, http.MethodGet,
		"/api/v1/verify/employee/EMP-001?kind=employment_certificate&serial="+serial+"&t="+token, "", nil)
	if okVerify.Code != http.StatusOK {
		t.Fatalf("تأیید سند باید ۲۰۰ بدهد، %d داد", okVerify.Code)
	}
	var docVerify map[string]any
	decodeInto(t, okVerify, &docVerify)
	if docVerify["documentVerified"] != true {
		t.Errorf("سند با امضای درست باید تأیید شود: %+v", docVerify)
	}
	badVerify := call(t, h, http.MethodGet,
		"/api/v1/verify/employee/EMP-001?kind=employment_certificate&serial="+serial+"&t=0123456789abcdef", "", nil)
	var badBody map[string]any
	decodeInto(t, badVerify, &badBody)
	if badBody["documentVerified"] == true {
		t.Errorf("امضای نادرست نباید سند را تأیید کند: %+v", badBody)
	}

	// ---------- حذف نشان ----------
	del := call(t, h, http.MethodDelete, "/api/v1/settings/companies/"+active.ID+"/logo", admin, nil)
	if del.Code != http.StatusOK {
		t.Fatalf("حذف نشان باید ۲۰۰ بدهد، %d داد: %s", del.Code, del.Body.String())
	}
	var deleted struct {
		Company contracts.Record `json:"company"`
	}
	decodeInto(t, del, &deleted)
	if deleted.Company.Fields["logoFile"] != "" {
		t.Errorf("پس از حذف، نشانی نشان باید پاک شود: %+v", deleted.Company.Fields)
	}
	if gone := call(t, h, http.MethodGet, fileURL, "", nil); gone.Code != http.StatusNotFound {
		t.Errorf("فایل نشان پس از حذف باید ۴۰۴ بدهد، %d داد", gone.Code)
	}
}

// keepJSONImport جلوگیری از حذف ناخواستهٔ import در ویرایش‌های بعدی (تجزیهٔ پاسخ در همین فایل).
var _ = json.Marshal
