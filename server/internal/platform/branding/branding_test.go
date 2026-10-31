// آزمون‌های انبار نشان شرکت: سنجش نوع فایل با امضای دودویی، پاک‌سازی نام، سرو و حذف.

package branding

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"strings"
	"testing"
)

// samplePNG یک تصویر PNG کوچک می‌سازد.
func samplePNG(t *testing.T) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 2, 2))
	img.Set(0, 0, color.RGBA{R: 30, G: 136, B: 229, A: 255})
	buf := &bytes.Buffer{}
	if err := png.Encode(buf, img); err != nil {
		t.Fatalf("ساخت تصویر آزمون ناموفق: %v", err)
	}
	return buf.Bytes()
}

func TestSaveOpenRemove(t *testing.T) {
	dir := t.TempDir()
	store, err := New(dir)
	if err != nil {
		t.Fatalf("ساخت انبار نباید خطا بدهد: %v", err)
	}
	data := samplePNG(t)
	file, err := store.Save("bipc", "logo.PNG", data)
	if err != nil {
		t.Fatalf("ذخیرهٔ نشان نباید خطا بدهد: %v", err)
	}
	if !strings.HasPrefix(file.Name, "bipc-") || !strings.HasSuffix(file.Name, ".png") {
		t.Errorf("نام فایل نشان باید کد شرکت و پسوند درست داشته باشد: %s", file.Name)
	}
	if file.ContentType != "image/png" || file.SizeBytes != len(data) || len(file.SHA256) != 16 {
		t.Errorf("اطلاعات فایل نشان درست نیست: %+v", file)
	}
	if file.URL != URL(file.Name) || !strings.HasPrefix(file.URL, "/api/v1/branding/") {
		t.Errorf("نشانی عمومی نشان درست نیست: %s", file.URL)
	}
	if !store.Exists(file.Name) {
		t.Errorf("فایل نشان باید پس از ذخیره موجود باشد")
	}
	handle, contentType, err := store.Open(file.Name)
	if err != nil {
		t.Fatalf("بازکردن فایل نشان نباید خطا بدهد: %v", err)
	}
	if contentType != "image/png" {
		t.Errorf("نوع محتوای فایل نشان باید image/png باشد، «%s» شد", contentType)
	}
	// در ویندوز تا وقتی دستهٔ فایل باز است حذف ممکن نیست؛ پس همان‌جا بسته می‌شود (سرو
	// هر درخواست هم همین کار را می‌کند).
	if err := handle.Close(); err != nil {
		t.Fatalf("بستن دستهٔ فایل نشان نباید خطا بدهد: %v", err)
	}
	if NameFromURL(file.URL) != file.Name {
		t.Errorf("بازخوانی نام از نشانی درست نیست: %s", NameFromURL(file.URL))
	}
	if err := store.Remove(file.Name); err != nil {
		t.Fatalf("حذف نشان نباید خطا بدهد: %v", err)
	}
	if store.Exists(file.Name) {
		t.Errorf("پس از حذف، فایل نشان نباید موجود باشد")
	}
	if err := store.Remove(file.Name); err != nil {
		t.Errorf("حذف دوبارهٔ فایلی که نیست نباید خطا بدهد: %v", err)
	}
}

func TestSaveRejectsBadInput(t *testing.T) {
	store, err := New(t.TempDir())
	if err != nil {
		t.Fatalf("ساخت انبار نباید خطا بدهد: %v", err)
	}
	cases := []struct {
		name  string
		owner string
		data  []byte
		why   string
	}{
		{"logo.png", "../etc", samplePNG(t), "کد شرکت ناامن"},
		{"logo.png", "bipc", nil, "فایل خالی"},
		{"note.txt", "bipc", []byte("سلام"), "نوع ناشناخته"},
		{"logo.svg", "bipc", []byte(`<svg><script>alert(1)</script></svg>`), "SVG با اسکریپت"},
		{"logo.png", "bipc", bytes.Repeat([]byte{0x89, 'P', 'N', 'G'}, MaxBytes), "حجم بیش از حد"},
	}
	for _, c := range cases {
		if _, err := store.Save(c.owner, c.name, c.data); err == nil {
			t.Errorf("%s باید رد شود", c.why)
		}
	}
	// SVG سالم پذیرفته می‌شود.
	if _, err := store.Save("bipc", "logo.svg", []byte(`<svg xmlns="http://www.w3.org/2000/svg" width="10" height="10"><rect width="10" height="10"/></svg>`)); err != nil {
		t.Errorf("SVG سالم باید پذیرفته شود: %v", err)
	}
}

func TestOpenRejectsUnsafeNames(t *testing.T) {
	store, err := New(t.TempDir())
	if err != nil {
		t.Fatalf("ساخت انبار نباید خطا بدهد: %v", err)
	}
	for _, name := range []string{"../secret.txt", "..\\secret.txt", "a/b.png", "", "خطر.png"} {
		if _, _, err := store.Open(name); err == nil {
			t.Errorf("نام ناامن «%s» باید رد شود", name)
		}
	}
}
