package contracts

import "testing"

// TestMergeManagementKeys قواعد سوار شدن کاتالوگ زمان‌اجرا روی فهرست پایه را می‌گیرد.
//
// این تابع پل میان «فهرست پایهٔ اعلامی در دامنه» و «اقلامی که کاربر در پنل تنظیمات
// می‌سازد» است؛ پس سه رفتار مهم دارد: بدون کاتالوگ رفتار پیش‌فرض عوض نشود، قلم تازه
// اضافه شود، و قلم غیرفعال کنار گذاشته شود.
func TestMergeManagementKeys(t *testing.T) {
	base := []Management{
		{Key: "bandar_imam", TitleFa: "بندر امام"},
		{Key: "kimia", TitleFa: "کیمیا"},
	}

	t.Run("بدون کاتالوگ همان فهرست پایه می‌ماند", func(t *testing.T) {
		got := MergeManagementKeys(base, nil)
		if len(got) != 2 || got["bandar_imam"] != "بندر امام" || got["kimia"] != "کیمیا" {
			t.Fatalf("فهرست پایه تغییر کرد: %#v", got)
		}
	})

	t.Run("قلم تازهٔ کاتالوگ اضافه می‌شود", func(t *testing.T) {
		got := MergeManagementKeys(base, []CatalogItem{
			{Key: "new_mgmt", TitleFa: "مدیریت تازه", Active: true},
		})
		if got["new_mgmt"] != "مدیریت تازه" {
			t.Fatalf("قلم تازه اضافه نشد: %#v", got)
		}
	})

	t.Run("عنوان قلم کاتالوگ بر پایه مقدم است", func(t *testing.T) {
		got := MergeManagementKeys(base, []CatalogItem{
			{Key: "kimia", TitleFa: "شرکت کیمیا (تازه)", Active: true},
		})
		if got["kimia"] != "شرکت کیمیا (تازه)" {
			t.Fatalf("عنوان زمان‌اجرا بر پایه مقدم نشد: %#v", got)
		}
	})

	t.Run("قلم غیرفعال از فهرست بیرون می‌رود", func(t *testing.T) {
		got := MergeManagementKeys(base, []CatalogItem{
			{Key: "kimia", Active: false},
		})
		if _, ok := got["kimia"]; ok {
			t.Fatalf("قلم غیرفعال حذف نشد: %#v", got)
		}
		if got["bandar_imam"] != "بندر امام" {
			t.Fatalf("غیرفعال کردن یک قلم، قلم دیگر را خراب کرد: %#v", got)
		}
	})

	t.Run("قلم بدون کلید نادیده گرفته می‌شود", func(t *testing.T) {
		got := MergeManagementKeys(base, []CatalogItem{{TitleFa: "بی‌کلید", Active: true}})
		if len(got) != 2 {
			t.Fatalf("قلم بی‌کلید نباید وارد فهرست شود: %#v", got)
		}
	})
}
