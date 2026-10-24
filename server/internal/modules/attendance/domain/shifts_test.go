// آزمون‌های جدولی دامنهٔ نوبت‌کاری: شکستن و سنجش چرخه، حل نوبت یک تاریخ و قواعد
// پذیرش تخصیص/تعویض. هر خطای دامنه باید از جنس RuleError باشد تا لایهٔ HTTP آن را
// ۴۲۲ کند و رابط کاربری پیام فارسی ببیند.

package domain

import (
	"errors"
	"strings"
	"testing"

	"hrms/internal/contracts"
	"hrms/internal/platform/calendar"
	"hrms/internal/platform/recordkit"
)

// mustBeRuleError می‌سنجد خطای دامنه از جنس Recordkit RuleError برگشته است.
func mustBeRuleError(t *testing.T, err error) {
	t.Helper()
	var rule *recordkit.RuleError
	if !errors.As(err, &rule) || rule.Message == "" {
		t.Fatalf("انتظار RuleError با پیام فارسی بود؛ %v آمد", err)
	}
}

func TestShiftParseCycle(t *testing.T) {
	cases := []struct {
		name    string
		raw     string
		want    []string
		wantErr bool
	}{
		{name: "جداشده با کامای لاتین", raw: "صبح,عصر,شب,استراحت",
			want: []string{"صبح", "عصر", "شب", "استراحت"}},
		{name: "جداشده با ویرگول فارسی و فاصله", raw: "صبح، عصر، شب",
			want: []string{"صبح", "عصر", "شب"}},
		{name: "جداکنندهٔ اسلش", raw: "شب/استراحت", want: []string{"شب", "استراحت"}},
		{name: "دو جایگاه حداقل است", raw: "شب,استراحت", want: []string{"شب", "استراحت"}},
		{name: "تک‌جایگاهی رد می‌شود", raw: "صبح", wantErr: true},
		{name: "کلید ناشناخته رد می‌شود", raw: "صبح,دوچرخه", wantErr: true},
		{name: "خالی رد می‌شود", raw: "   ", wantErr: true},
		{name: "فقط جداکننده رد می‌شود", raw: "، ،", wantErr: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ParseCycle(tc.raw)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("باید خطا می‌داد، گرفت: %v", got)
				}
				mustBeRuleError(t, err)
				return
			}
			if err != nil {
				t.Fatalf("نباید خطا می‌داد: %v", err)
			}
			if len(got) != len(tc.want) {
				t.Fatalf("تعداد جایگاه‌ها %d است، %d انتظار می‌رفت: %v", len(got), len(tc.want), got)
			}
			for i, want := range tc.want {
				if got[i] != want {
					t.Errorf("جایگاه %d باید «%s» باشد، «%s» شد", i, want, got[i])
				}
			}
		})
	}
}

func TestShiftResolveDay(t *testing.T) {
	cycle := []string{"صبح", "عصر", "شب", "استراحت"}
	cases := []struct {
		name    string
		cycle   []string
		start   string
		offset  int
		day     string
		wantKey string
		wantIdx int
		wantErr bool
	}{
		{name: "روز آغاز همان روز شروع", cycle: cycle, start: "1404/07/01", offset: 0,
			day: "1404/07/01", wantKey: "صبح", wantIdx: 0},
		{name: "روز بعد جایگاه بعدی چرخه", cycle: cycle, start: "1404/07/01", offset: 0,
			day: "1404/07/02", wantKey: "عصر", wantIdx: 1},
		{name: "پیچیدن چرخه پس از یک دور", cycle: cycle, start: "1404/07/01", offset: 0,
			day: "1404/07/05", wantKey: "صبح", wantIdx: 0},
		{name: "جایگاه آغاز غیر صفر", cycle: cycle, start: "1404/07/01", offset: 2,
			day: "1404/07/01", wantKey: "شب", wantIdx: 2},
		{name: "جمع جایگاه آغاز و فاصلهٔ روز", cycle: cycle, start: "1404/07/01", offset: 3,
			day: "1404/07/03", wantKey: "عصر", wantIdx: 1},
		{name: "عبور از مرز ماه شمسی", cycle: cycle, start: "1404/06/31", offset: 0,
			day: "1404/07/01", wantKey: "عصر", wantIdx: 1},
		{name: "پیش از مبدأ تخصیص خطا می‌دهد", cycle: cycle, start: "1404/07/01", offset: 0,
			day: "1404/06/30", wantErr: true},
		{name: "چرخهٔ خالی خطا می‌دهد", cycle: nil, start: "1404/07/01", offset: 0,
			day: "1404/07/01", wantErr: true},
		{name: "تاریخ نامعتبر خطا می‌دهد", cycle: cycle, start: "1404/07/01", offset: 0,
			day: "فردا", wantErr: true},
		{name: "شروع نامعتبر خطا می‌دهد", cycle: cycle, start: "دیروز", offset: 0,
			day: "1404/07/01", wantErr: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			key, idx, err := ResolveShiftDay(tc.cycle, tc.start, tc.offset, tc.day)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("باید خطا می‌داد، گرفت: (%s, %d)", key, idx)
				}
				mustBeRuleError(t, err)
				return
			}
			if err != nil {
				t.Fatalf("نباید خطا می‌داد: %v", err)
			}
			if key != tc.wantKey || idx != tc.wantIdx {
				t.Fatalf(" (%s, %d) گرفت؛ (%s, %d) انتظار می‌رفت", key, idx, tc.wantKey, tc.wantIdx)
			}
		})
	}
}

// TestShiftResolveDayCoversFullCycle می‌سنجد یک دور کامل چرخه دقیقاً همان ترتیب را
// بسازد و پس از آن تکرار شود — تضمین هم‌پوشانی تخصیص‌ها در روزهای بعد.
func TestShiftResolveDayCoversFullCycle(t *testing.T) {
	cycle := []string{"صبح", "عصر", "شب", "استراحت"}
	start, err := calendar.Parse("1404/07/01")
	if err != nil {
		t.Fatalf("تاریخ نمونه نامعتبر: %v", err)
	}
	var seen []string
	for i := 0; i < 8; i++ {
		day, err := start.AddDays(i)
		if err != nil {
			t.Fatalf("افزودن روز ناموفق: %v", err)
		}
		key, _, err := ResolveShiftDay(cycle, start.String(), 0, day.String())
		if err != nil {
			t.Fatalf("روز %d نباید خطا می‌داد: %v", i, err)
		}
		seen = append(seen, key)
	}
	want := "صبح,عصر,شب,استراحت,صبح,عصر,شب,استراحت"
	if got := strings.Join(seen, ","); got != want {
		t.Fatalf("دو دور چرخه نادرست است: %s", got)
	}
}

func TestShiftPatternRules(t *testing.T) {
	cases := []struct {
		name    string
		fields  map[string]string
		wantErr bool
	}{
		{name: "چرخهٔ معتبر با حداقل پوشش", fields: map[string]string{
			"rotation": "صبح,عصر,شب,استراحت", "minCoverage": "2"}},
		{name: "بدون حداقل پوشش مجاز است", fields: map[string]string{
			"rotation": "صبح,استراحت"}},
		{name: "چرخهٔ تکی رد می‌شود", fields: map[string]string{
			"rotation": "صبح"}, wantErr: true},
		{name: "کلید ناشناخته رد می‌شود", fields: map[string]string{
			"rotation": "شب,تعطیل"}, wantErr: true},
		{name: "پوشش منفی رد می‌شود", fields: map[string]string{
			"rotation": "صبح,عصر", "minCoverage": "-1"}, wantErr: true},
		{name: "پوشش کسری رد می‌شود", fields: map[string]string{
			"rotation": "صبح,عصر", "minCoverage": "1.5"}, wantErr: true},
		{name: "پوشش غیرعددی رد می‌شود", fields: map[string]string{
			"rotation": "صبح,عصر", "minCoverage": "دو"}, wantErr: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateShiftPattern(contracts.RecordInput{Kind: "shiftPatterns", Fields: tc.fields})
			if tc.wantErr {
				if err == nil {
					t.Fatal("باید رد می‌شد ولی پذیرفته شد")
				}
				mustBeRuleError(t, err)
				return
			}
			if err != nil {
				t.Fatalf("نباید رد می‌شد: %v", err)
			}
		})
	}
}

func TestShiftAssignmentRules(t *testing.T) {
	cases := []struct {
		name    string
		fields  map[string]string
		wantErr bool
	}{
		{name: "تخصیص بدون تاریخ پایان", fields: map[string]string{
			"pattern": "الگوی نمونه", "startDate": "1404/07/01"}},
		{name: "پایان هم‌زمان شروع مجاز است", fields: map[string]string{
			"pattern": "الگوی نمونه", "startDate": "1404/07/01", "endDate": "1404/07/01"}},
		{name: "پایان پس از شروع", fields: map[string]string{
			"pattern": "الگوی نمونه", "startDate": "1404/07/01", "endDate": "1404/07/20",
			"cycleOffset": "3"}},
		{name: "پایان پیش از شروع رد می‌شود", fields: map[string]string{
			"pattern": "الگوی نمونه", "startDate": "1404/07/10", "endDate": "1404/07/01"},
			wantErr: true},
		{name: "جایگاه منفی رد می‌شود", fields: map[string]string{
			"pattern": "الگوی نمونه", "startDate": "1404/07/01", "cycleOffset": "-1"},
			wantErr: true},
		{name: "جایگاه غیرعددی رد می‌شود", fields: map[string]string{
			"pattern": "الگوی نمونه", "startDate": "1404/07/01", "cycleOffset": "یک"},
			wantErr: true},
		{name: "پایان نامعتبر رد می‌شود", fields: map[string]string{
			"pattern": "الگوی نمونه", "startDate": "1404/07/01", "endDate": "فردا"},
			wantErr: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateShiftAssignment(contracts.RecordInput{Kind: "shiftAssignments", Fields: tc.fields})
			if tc.wantErr {
				if err == nil {
					t.Fatal("باید رد می‌شد ولی پذیرفته شد")
				}
				mustBeRuleError(t, err)
				return
			}
			if err != nil {
				t.Fatalf("نباید رد می‌شد: %v", err)
			}
		})
	}
}

func TestShiftSwapRules(t *testing.T) {
	cases := []struct {
		name     string
		employee string
		fields   map[string]string
		wantErr  bool
	}{
		{name: "تعویض معتبر با دو شیفت متفاوت", employee: "EMP-005",
			fields: map[string]string{"swapDate": "1404/07/10", "partnerEmployee": "EMP-008",
				"myShift": "صبح", "partnerShift": "عصر"}},
		{name: "شیفت‌ها اختیاری‌اند", employee: "EMP-005",
			fields: map[string]string{"swapDate": "1404/07/10", "partnerEmployee": "EMP-008"}},
		{name: "همتا خودِ درخواست‌دهنده رد می‌شود", employee: "EMP-005",
			fields:  map[string]string{"swapDate": "1404/07/10", "partnerEmployee": "EMP-005"},
			wantErr: true},
		{name: "تعویض هم‌شیفت بی‌اثر و رد می‌شود", employee: "EMP-005",
			fields: map[string]string{"swapDate": "1404/07/10", "partnerEmployee": "EMP-008",
				"myShift": "شب", "partnerShift": "شب"},
			wantErr: true},
		{name: "کلید شیفت ناشناخته در سمت من رد می‌شود", employee: "EMP-005",
			fields: map[string]string{"swapDate": "1404/07/10", "partnerEmployee": "EMP-008",
				"myShift": "دوچرخه", "partnerShift": "عصر"},
			wantErr: true},
		{name: "کلید شیفت ناشناخته در سمت مقابل رد می‌شود", employee: "EMP-005",
			fields: map[string]string{"swapDate": "1404/07/10", "partnerEmployee": "EMP-008",
				"myShift": "صبح", "partnerShift": "نیمه‌شب"},
			wantErr: true},
		{name: "تاریخ نامعتبر رد می‌شود", employee: "EMP-005",
			fields:  map[string]string{"swapDate": "پس‌فردا", "partnerEmployee": "EMP-008"},
			wantErr: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateShiftSwap(contracts.RecordInput{
				Kind: "shiftSwaps", Title: "تست", EmployeeID: tc.employee, Fields: tc.fields,
			})
			if tc.wantErr {
				if err == nil {
					t.Fatal("باید رد می‌شد ولی پذیرفته شد")
				}
				mustBeRuleError(t, err)
				return
			}
			if err != nil {
				t.Fatalf("نباید رد می‌شد: %v", err)
			}
		})
	}
}
