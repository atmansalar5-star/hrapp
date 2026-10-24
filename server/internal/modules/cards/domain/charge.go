// Package domain دامنهٔ ماژول کارت‌های رفاهی و شارژ.
//
// این بسته تنها منبع حقیقت شارژ است: بخش‌ها (فرهنگی، رستوران، پاداش تحصیلی و چهار
// قلم رفاهی نفت‌کارت)، نرخ پیش‌فرض هر بخش، قاعدهٔ تخصیص به افراد تحت تکفل و موتور
// ساخت ردیف‌های شارژ. UI و API هیچ برچسب یا قاعده‌ای را از خودشان نمی‌سازند.
package domain

import (
	"sort"
	"strings"
)

// شناسهٔ کارت‌های رفاهی. نفت‌کارت همان کارتی است که در سازمان با نام‌های
// «نفت‌مارت»، «فارس‌کارت» و «سامان‌کارت» شناخته می‌شود.
const (
	CardPetrokart = "petrokart"
	CardNaftmart  = "naftmart"
)

// Section یک بخش یا قلم شارژ روی کارت رفاهی.
//
// PerDependent یعنی مبلغ ماهیانه «به ازای هر نفر» تعلق می‌گیرد؛ پس با تعریف هر فرد
// مشمول در پنل تخصیص، شارژ همان فرد ساخته می‌شود.
type Section struct {
	Key          string
	TitleFa      string
	Card         string
	PerDependent bool
	DefaultRate  int64
	UnitFa       string
}

// Sections فهرست کامل بخش‌های شارژ — تنها منبع حقیقت.
func Sections() []Section {
	return []Section{
		{Key: "petrokart_cultural", TitleFa: "فرهنگی (پتروکارت)", Card: CardPetrokart, PerDependent: true, DefaultRate: 5_000_000, UnitFa: "ماهانه به ازای هر نفر"},
		{Key: "petrokart_restaurant", TitleFa: "رستوران (پتروکارت)", Card: CardPetrokart, PerDependent: true, DefaultRate: 8_000_000, UnitFa: "ماهانه به ازای هر نفر"},
		{Key: "petrokart_education", TitleFa: "پاداش تحصیلی فرزندان ممتاز", Card: CardPetrokart, PerDependent: true, DefaultRate: 12_000_000, UnitFa: "ماهانه به ازای هر فرزند ممتاز"},
		{Key: "naftmart_food", TitleFa: "کالای اساسی و خواروبار", Card: CardNaftmart, PerDependent: true, DefaultRate: 10_000_000, UnitFa: "ماهانه به ازای هر نفر"},
		{Key: "naftmart_protein", TitleFa: "گوشت، مرغ و پروتئین", Card: CardNaftmart, PerDependent: true, DefaultRate: 6_000_000, UnitFa: "ماهانه به ازای هر نفر"},
		{Key: "naftmart_dairy", TitleFa: "لبنیات", Card: CardNaftmart, PerDependent: true, DefaultRate: 3_000_000, UnitFa: "ماهانه به ازای هر نفر"},
		{Key: "naftmart_hygiene", TitleFa: "شوینده و بهداشتی", Card: CardNaftmart, PerDependent: true, DefaultRate: 2_500_000, UnitFa: "ماهانه به ازای هر نفر"},
	}
}

// CardTitleFa عنوان فارسی یک کارت.
func CardTitleFa(card string) string {
	switch card {
	case CardPetrokart:
		return "پتروکارت"
	case CardNaftmart:
		return "نفت‌کارت (نفت‌مارت / فارس‌کارت / سامان‌کارت)"
	}
	return card
}

// ValidCard بررسی معتبر بودن شناسهٔ کارت.
func ValidCard(card string) bool {
	return card == CardPetrokart || card == CardNaftmart
}

// Cards فهرست کارت‌های پشتیبانی‌شده.
func Cards() []string { return []string{CardPetrokart, CardNaftmart} }

// SectionByKey یافتن یک بخش با کلید.
func SectionByKey(key string) (Section, bool) {
	for _, s := range Sections() {
		if s.Key == key {
			return s, true
		}
	}
	return Section{}, false
}

// SectionsOf بخش‌های یک کارت به ترتیب اعلام.
func SectionsOf(card string) []Section {
	out := []Section{}
	for _, s := range Sections() {
		if s.Card == card {
			out = append(out, s)
		}
	}
	return out
}

// AllocationField فیلد تخصیص در رکورد «فرد مشمول» که به این بخش گره می‌خورد.
//
// هر بخش یک کلید تخصیص دارد و چهار قلم نفت‌کارت یک کلید مشترک، چون در عمل یک
// تخصیص نفت‌کارت همهٔ اقلام چهارگانه را فعال می‌کند.
func AllocationField(sectionKey string) string {
	switch sectionKey {
	case "petrokart_cultural":
		return "petrokartCultural"
	case "petrokart_restaurant":
		return "petrokartRestaurant"
	case "petrokart_education":
		return "educationBonus"
	case "naftmart_food", "naftmart_protein", "naftmart_dairy", "naftmart_hygiene":
		return "naftmart"
	}
	return ""
}

// EducationSectionKey بخش پاداش تحصیلی؛ فقط برای فرزند ممتاز شارژ می‌شود.
const EducationSectionKey = "petrokart_education"

// Beneficiary یک فرد مشمول شارژ: خود کارمند یا فرد تحت تکفل او.
type Beneficiary struct {
	RecordID     string
	EmployeeID   string
	EmployeeName string
	Name         string
	Relation     string
	Excellent    bool
	Allocated    map[string]bool
}

// OccasionCharge مناسبت تقویمی با مبلغ شارژ مناسبتی.
type OccasionCharge struct {
	ID      string
	TitleFa string
	DateFa  string
	Amount  int64
	Card    string
}

// ChargeInput ورودی ساخت ردیف‌های شارژ.
type ChargeInput struct {
	Card          string
	Sections      []string // خالی یعنی همهٔ بخش‌های همین کارت
	RateBySection map[string]int64
	Beneficiaries []Beneficiary
	Occasion      *OccasionCharge
	Period        string
}

// ChargeLine یک ردیف شارژ برای یک ذی‌نفع در یک بخش.
type ChargeLine struct {
	EmployeeID   string
	EmployeeName string
	Beneficiary  string
	Relation     string
	Section      string
	SectionFa    string
	OccasionFa   string
	Amount       int64
	BasisFa      string
}

// DefaultRates نرخ‌های پیش‌فرض همهٔ بخش‌ها (برای بذرکاری رکوردهای نرخ).
func DefaultRates() map[string]int64 {
	out := map[string]int64{}
	for _, s := range Sections() {
		out[s.Key] = s.DefaultRate
	}
	return out
}

// Build ردیف‌های شارژ را از روی تخصیص افراد و نرخ‌های جاری می‌سازد.
//
// قواعد:
//   - فقط بخش‌های کارتِ درخواستی ساخته می‌شوند؛ بخشی از کارت دیگر هشدار می‌دهد.
//   - هر فرد مشمول فقط برای بخش‌هایی که در پنل تخصیص به او داده شده شارژ می‌شود.
//   - بخش پاداش تحصیلی بدون نشانهٔ «فرزند ممتاز» شارژ نمی‌شود (هشدار می‌دهد).
//   - نرخ نبودن یا صفر بودن به نرخ پیش‌فرض برمی‌گردد؛ اگر آن هم صفر باشد ردیف ساخته نمی‌شود.
//   - مناسبت انتخابی به‌ازای هر فرد مشمول یک ردیف جدا می‌سازد.
func Build(in ChargeInput) ([]ChargeLine, []string) {
	warnings := []string{}
	card := strings.TrimSpace(in.Card)
	if !ValidCard(card) {
		return nil, []string{"کارت انتخاب‌شده نامعتبر است."}
	}

	allowed := SectionsOf(card)
	chosen := allowed
	if len(in.Sections) > 0 {
		chosen = nil
		for _, key := range in.Sections {
			key = strings.TrimSpace(key)
			sec, ok := SectionByKey(key)
			if !ok {
				warnings = append(warnings, "بخش ناشناخته نادیده گرفته شد: "+key)
				continue
			}
			if sec.Card != card {
				warnings = append(warnings, "بخش «"+sec.TitleFa+"» به این کارت تعلق ندارد و نادیده گرفته شد.")
				continue
			}
			chosen = append(chosen, sec)
		}
		if len(chosen) == 0 {
			warnings = append(warnings, "هیچ بخشی از این کارت انتخاب نشده است.")
		}
	}

	rate := func(sec Section) (int64, bool) {
		if v, ok := in.RateBySection[sec.Key]; ok && v > 0 {
			return v, true
		}
		if sec.DefaultRate > 0 {
			warnings = append(warnings, "نرخ جاری «"+sec.TitleFa+"» تعریف نشده است؛ نرخ پیش‌فرض به‌کار رفت.")
			return sec.DefaultRate, true
		}
		warnings = append(warnings, "نرخ «"+sec.TitleFa+"» تعریف نشده است؛ ردیفی ساخته نشد.")
		return 0, false
	}

	lines := make([]ChargeLine, 0, len(in.Beneficiaries)*len(chosen))
	for _, b := range in.Beneficiaries {
		name := strings.TrimSpace(b.Name)
		if name == "" {
			name = b.EmployeeName
		}
		for _, sec := range chosen {
			if field := AllocationField(sec.Key); field != "" && !b.Allocated[field] {
				continue
			}
			if sec.Key == EducationSectionKey && !b.Excellent {
				warnings = append(warnings, "پاداش تحصیلی برای «"+name+"» تعلق نگرفت: نشانهٔ فرزند ممتاز ثبت نشده است.")
				continue
			}
			amount, ok := rate(sec)
			if !ok {
				continue
			}
			lines = append(lines, ChargeLine{
				EmployeeID:   b.EmployeeID,
				EmployeeName: b.EmployeeName,
				Beneficiary:  name,
				Relation:     b.Relation,
				Section:      sec.Key,
				SectionFa:    sec.TitleFa,
				Amount:       amount,
				BasisFa:      sec.UnitFa,
			})
		}
		if occ := in.Occasion; occ != nil && occ.Amount > 0 && (occ.Card == "" || occ.Card == card) {
			lines = append(lines, ChargeLine{
				EmployeeID:   b.EmployeeID,
				EmployeeName: b.EmployeeName,
				Beneficiary:  name,
				Relation:     b.Relation,
				Section:      "occasion",
				SectionFa:    "شارژ مناسبتی",
				OccasionFa:   occ.TitleFa,
				Amount:       occ.Amount,
				BasisFa:      "مناسبت " + occ.TitleFa + " (" + occ.DateFa + ")",
			})
		}
	}

	sort.SliceStable(lines, func(i, j int) bool {
		if lines[i].EmployeeID != lines[j].EmployeeID {
			return lines[i].EmployeeID < lines[j].EmployeeID
		}
		if lines[i].Beneficiary != lines[j].Beneficiary {
			return lines[i].Beneficiary < lines[j].Beneficiary
		}
		return lines[i].Section < lines[j].Section
	})
	return lines, warnings
}

// SumLines جمع مبالغ ردیف‌ها و جمع هر بخش.
func SumLines(lines []ChargeLine) (int64, map[string]int64, map[string]int) {
	total := int64(0)
	bySection := map[string]int64{}
	count := map[string]int{}
	for _, l := range lines {
		total += l.Amount
		bySection[l.Section] += l.Amount
		count[l.Section]++
	}
	return total, bySection, count
}
