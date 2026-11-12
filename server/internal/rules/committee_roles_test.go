package rules

import "testing"

// TestVeteranCommitteeRolesAreCompleteAndOrdered نقش‌ها کلید یکتا و ترتیب صعودی دارند.
func TestVeteranCommitteeRolesAreCompleteAndOrdered(t *testing.T) {
	roles := VeteranCommitteeRoles()
	if len(roles) < 6 {
		t.Fatalf("دست‌کم شش نقش برای کمیته لازم است، بود: %d", len(roles))
	}
	seen := map[string]bool{}
	last := 0
	for _, role := range roles {
		if role.Key == "" || role.TitleFa == "" {
			t.Fatalf("نقش بی‌کلید یا بی‌عنوان: %+v", role)
		}
		if seen[role.Key] {
			t.Fatalf("نقش تکراری: %s", role.Key)
		}
		seen[role.Key] = true
		if role.Order <= last {
			t.Fatalf("ترتیب نقش‌ها باید صعودی باشد: %s", role.Key)
		}
		last = role.Order
	}
	if !seen[VeteranCommitteeChairman] || !seen[VeteranCommitteeSecretary] {
		t.Fatal("رئیس و دبیر کمیته باید در فهرست نقش‌ها باشند")
	}
}

// TestVeteranCommitteeRoleOfFallsBack نقش ناشناخته به «عضو کمیته» برمی‌گردد.
func TestVeteranCommitteeRoleOfFallsBack(t *testing.T) {
	if got := VeteranCommitteeRoleOf(""); got != VeteranCommitteeMember {
		t.Fatalf("نقش خالی باید عضو کمیته شود، بود: %s", got)
	}
	if got := VeteranCommitteeRoleOf("  king "); got != VeteranCommitteeMember {
		t.Fatalf("نقش ناشناخته باید عضو کمیته شود، بود: %s", got)
	}
	if got := VeteranCommitteeRoleOf(VeteranCommitteeChairman); got != VeteranCommitteeChairman {
		t.Fatalf("نقش معلوم باید حفظ شود، بود: %s", got)
	}
	if got := VeteranCommitteeRoleTitleFa("king"); got != "عضو کمیته" {
		t.Fatalf("عنوان نقش ناشناخته باید «عضو کمیته» باشد، بود: %s", got)
	}
	if Assert := VeteranCommitteeRoleOrder(VeteranCommitteeChairman); Assert >= VeteranCommitteeRoleOrder(VeteranCommitteeMember) {
		t.Fatal("رئیس کمیته باید پیش از عضو عادی امضا کند")
	}
}

// TestVeteranCommitteeSignatureFa عنوان امضا از نقش و نام ساخته می‌شود.
func TestVeteranCommitteeSignatureFa(t *testing.T) {
	if got := VeteranCommitteeSignatureFa(VeteranCommitteeChairman, "سارا محمدی", ""); got != "رئیس کمیته — سارا محمدی" {
		t.Fatalf("عنوان امضای رئیس نادرست: %s", got)
	}
	// عنوان دلخواه بر عنوان نقش مقدم است.
	if got := VeteranCommitteeSignatureFa(VeteranCommitteeFoundation, "مهدی رحیمی", "نمایندهٔ بنیاد شهید"); got != "نمایندهٔ بنیاد شهید — مهدی رحیمی" {
		t.Fatalf("عنوان دلخواه باید مقدم باشد: %s", got)
	}
	// بی‌نام، فقط عنوان نقش می‌ماند.
	if got := VeteranCommitteeSignatureFa(VeteranCommitteeSecretary, "  ", ""); got != "دبیر کمیته" {
		t.Fatalf("امضای بی‌نام باید فقط نقش باشد: %s", got)
	}
}

// TestVeteranCommitteeSignOffFa متن امضاکنندگان و تاریخ.
func TestVeteranCommitteeSignOffFa(t *testing.T) {
	if got := VeteranCommitteeSignOffFa(nil, "1405/08/20"); got != "" {
		t.Fatalf("بی‌امضا باید متن خالی بدهد، بود: %s", got)
	}
	got := VeteranCommitteeSignOffFa([]string{"رئیس کمیته — سارا محمدی", " ", "دبیر کمیته — مریم نوری"}, "1405/08/20")
	want := "امضا شدگان: رئیس کمیته — سارا محمدی، دبیر کمیته — مریم نوری — تاریخ امضا 1405/08/20"
	if got != want {
		t.Fatalf("متن امضاکنندگان نادرست:\nبود: %s\nباید: %s", got, want)
	}
	if got := VeteranCommitteeSignOffFa([]string{"رئیس کمیته"}, ""); got != "امضا شدگان: رئیس کمیته" {
		t.Fatalf("بدون تاریخ نباید تاریخ درج شود: %s", got)
	}
}
