// تعرفه‌های کمک هزینهٔ درمانی ۱۴۰۵ — خواندن دادهٔ برون‌استخراج‌شده و تبدیل به رکورد.
//
// چرا دادهٔ JSON و جدا از کد: تعرفه‌های سالانه یک «دادهٔ مصوب» است، نه منطق برنامه؛ نگه‌داشتنش
// در فایل داده این امکان را می‌دهد که بازبینی چشمی، مقایسه با دستورالعمل و بازتولید بدون
// دست‌زدن به کد انجام شود. کد فقط آن را به رکوردهای موتور `recordkit` تبدیل می‌کند و از آن پس
// هر تغییر تعرفه از رابط سامانه انجام می‌گیرد (تعرفه، دادهٔ پارامتریک است).
package domain

import (
	"embed"
	"encoding/json"
	"strconv"
	"strings"

	"hrms/internal/contracts"
)

//go:embed tariffs_1405.json
var tariffFiles embed.FS

// tariffSeedRow یک ردیف تعرفهٔ برون‌استخراج‌شده از دستورالعمل ۱۴۰۵.
type tariffSeedRow struct {
	Code                string  `json:"code"`
	Category            string  `json:"category"`
	TitleFa             string  `json:"titleFa"`
	UnitFa              string  `json:"unitFa,omitempty"`
	Amount              int64   `json:"amount"`
	CeilingAmount       int64   `json:"ceilingAmount,omitempty"`
	OrgSharePercent     float64 `json:"orgSharePercent,omitempty"`
	RequirePrescription bool    `json:"requirePrescription,omitempty"`
	EffectiveFrom       string  `json:"effectiveFrom"`
	RefFa               string  `json:"refFa,omitempty"`
	NoteFa              string  `json:"noteFa,omitempty"`
}

// tariffSeedFile ساختار فایل داده.
type tariffSeedFile struct {
	Source    string          `json:"source"`
	Effective string          `json:"effectiveFrom"`
	Tariffs   []tariffSeedRow `json:"tariffs"`
}

// TariffSeed1405 ردیف‌های تعرفهٔ ۱۴۰۵ را به شکل رکورد قابل بذرکاری برمی‌گرداند.
//
// قاعدهٔ بذر: هر ردیف یک رکورد «تعرفهٔ کمک هزینهٔ درمانی» با وضعیت آغازین پیش‌نویس است (پس از
// اجرا، فهرست تعرفه‌ها یک‌بار از همین‌جا پر می‌شود)؛ اگر فایل داده ناخوانا باشد، بذرکاری
// بی‌صدا رد می‌شود و سامانه با فهرست خالی بالا می‌آید (نبود تعرفه نباید راه‌اندازی را بشکند).
func TariffSeed1405() []contracts.RecordInput {
	raw, err := tariffFiles.ReadFile("tariffs_1405.json")
	if err != nil {
		return nil
	}
	var file tariffSeedFile
	if err := json.Unmarshal(raw, &file); err != nil {
		return nil
	}
	out := make([]contracts.RecordInput, 0, len(file.Tariffs))
	for _, row := range file.Tariffs {
		effective := strings.TrimSpace(row.EffectiveFrom)
		if effective == "" {
			effective = file.Effective
		}
		fields := map[string]string{
			"code":          strings.TrimSpace(row.Code),
			"category":      strings.TrimSpace(row.Category),
			"unitFa":        strings.TrimSpace(row.UnitFa),
			"amount":        strconv.FormatInt(row.Amount, 10),
			"effectiveFrom": effective,
			"refFa":         strings.TrimSpace(row.RefFa),
		}
		if row.CeilingAmount > 0 {
			fields["ceilingAmount"] = strconv.FormatInt(row.CeilingAmount, 10)
		}
		if row.OrgSharePercent > 0 {
			// درصد در فیلد رکورد به‌صورت «۷۰» نگه داشته می‌شود و در محاسبه به نسبت تبدیل می‌شود.
			fields["orgSharePercent"] = strconv.FormatFloat(row.OrgSharePercent, 'f', -1, 64)
		}
		if row.RequirePrescription {
			fields["requirePrescription"] = "yes"
		}
		out = append(out, contracts.RecordInput{
			Kind:   KindTariffs,
			Title:  strings.TrimSpace(row.TitleFa),
			Fields: fields,
			NoteFa: strings.TrimSpace(row.NoteFa),
			// تعرفه‌های دستورالعمل ۱۴۰۵ «مصوب و جاری» هستند؛ اگر پیش‌نویس بذر شوند، موتور
			// محاسبه آن‌ها را «تعرفهٔ جاری» نمی‌شناسد و هیچ درخواستی پرداخت نمی‌شود.
			SeedStatus: "active",
		})
	}
	return out
}
