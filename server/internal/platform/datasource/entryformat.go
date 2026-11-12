// قالب ورود اطلاعات دستی — الگو و نمونهٔ هر ستون فرم و گزارش.
//
// چرا در همین بستهٔ ساختار داده: قالب اکسل و راهنمای بارگذاری از همین ساختار ساخته
// می‌شود؛ «ورود دستی» هم همان ستون‌ها را لازم دارد. اگر قالب دستی جای دیگری تعریف شود،
// دو توضیح ناهمخوان از یک جدول پیدا می‌شود.
package datasource

import (
	"strings"

	"hrms/internal/contracts"
)

// FieldFormatFa الگوی ورودی یک نوع فیلد به فارسی (برای نمایش در قالب دستی).
//
// همان متن قالبی است که راهنمای اکسل به کار می‌برد، ولی از بیرون بسته هم قابل خواندن
// است تا مسیر گزارش‌ها و رابط کاربری، الگو را از یک جا بگیرند.
func FieldFormatFa(fieldType string) string { return formatOf(fieldType) }

// FieldExample نمونهٔ یک مقدار درست برای همان نوع فیلد.
//
// نمونه‌ها عمداً «واقع‌نما» ولی بی‌اثرند (تاریخ و شمارهٔ ساختگی) تا کاربر با نگاه به
// برگهٔ قالب بفهمد مقدار را چه شکلی وارد کند، بدون آنکه عددی از سازمان در سند پخش شود.
func FieldExample(fieldType, labelFa string) string {
	switch fieldType {
	case TypeDate:
		return "1405/08/12"
	case TypeMoney:
		return "185000000"
	case TypeNumber:
		return "12"
	case TypeSelect:
		return "یکی از مقادیر مجاز همان ستون"
	}
	if strings.TrimSpace(labelFa) != "" {
		return "متن آزاد — " + strings.TrimSpace(labelFa)
	}
	return "متن آزاد"
}

// EntryFormat یک ردیف از برگهٔ «قالب ورود دستی».
type EntryFormat struct {
	// Column شمارهٔ ستون در برگهٔ ورودی (از ۱).
	Column   int    `json:"column"`
	Key      string `json:"key"`
	LabelFa  string `json:"labelFa"`
	Type     string `json:"type"`
	TypeFa   string `json:"typeFa"`
	Required bool   `json:"required"`
	// FormatFa الگوی ورود (مثل YYYY/MM/DD یا «عدد بدون جداکننده»).
	FormatFa string `json:"formatFa,omitempty"`
	// ValuesFa مقادیر مجاز فیلدهای انتخابی.
	ValuesFa []string `json:"valuesFa,omitempty"`
	// Example یک مقدار نمونهٔ درست.
	Example string `json:"example,omitempty"`
	NoteFa  string `json:"noteFa,omitempty"`
}

// EntryFormatFromEntity برگهٔ قالب ورود دستی یک جدول/فرم را از ساختار همان جدول می‌سازد.
func EntryFormatFromEntity(entity Entity) []EntryFormat {
	out := make([]EntryFormat, 0, len(entity.Fields))
	for i, field := range entity.Fields {
		format := field.FormatFa
		if strings.TrimSpace(format) == "" {
			format = formatOf(field.Type)
		}
		example := field.Example
		if strings.TrimSpace(example) == "" {
			example = FieldExample(field.Type, field.TitleFa)
		}
		note := field.NoteFa
		if strings.TrimSpace(note) == "" && field.LookupFa != "" {
			note = "ارجاع به: " + field.LookupFa
		}
		out = append(out, EntryFormat{
			Column: i + 1, Key: field.Key, LabelFa: field.TitleFa,
			Type: field.Type, TypeFa: typeFa(field.Type), Required: field.Required,
			FormatFa: format, ValuesFa: field.ValuesFa, Example: example, NoteFa: note,
		})
	}
	return out
}

// EntryFormatFromReportFields برگهٔ قالب ورود دستی ستون‌های یک منبع گزارش.
//
// چرا reports هم قالب می‌گیرد: بخشی از گزارش‌ها با ورودی دستی کار می‌کنند (فیلترها و
// ستون‌های انتخابی)؛ برگهٔ قالب می‌گوید هر ستون چه نوعی دارد و مقدار را چه شکلی بنویسیم.
func EntryFormatFromReportFields(fields []contracts.ReportField) []EntryFormat {
	out := make([]EntryFormat, 0, len(fields))
	for i, field := range fields {
		out = append(out, EntryFormat{
			Column: i + 1, Key: field.Key, LabelFa: field.LabelFa,
			Type: field.Type, TypeFa: typeFa(field.Type), Required: false,
			FormatFa: formatOf(field.Type), Example: FieldExample(field.Type, field.LabelFa),
		})
	}
	return out
}

// SampleRow یک سطر نمونه از برگهٔ قالب می‌سازد (همان الگوی هر ستون).
func SampleRow(format []EntryFormat) []string {
	out := make([]string, 0, len(format))
	for _, col := range format {
		out = append(out, col.Example)
	}
	return out
}

// EntrySheet برگهٔ کامل قالب ورود دستی یک جدول: سرصفحه، ستون‌ها و سطر نمونه.
type EntrySheet struct {
	Key       string        `json:"key"`
	TitleFa   string        `json:"titleFa"`
	ModuleFa  string        `json:"moduleFa,omitempty"`
	GroupFa   string        `json:"groupFa,omitempty"`
	SourceFa  string        `json:"sourceFa,omitempty"`
	Format    []EntryFormat `json:"format"`
	SampleRow []string      `json:"sampleRow"`
	NoteFa    string        `json:"noteFa,omitempty"`
}

// EntrySheetOf برگهٔ قالب یک جدول فرم را می‌سازد.
func EntrySheetOf(entity Entity) EntrySheet {
	format := EntryFormatFromEntity(entity)
	return EntrySheet{
		Key: entity.Key, TitleFa: entity.TitleFa, ModuleFa: entity.ModuleFa,
		GroupFa: entity.GroupFa, SourceFa: entity.SourceFa,
		Format: format, SampleRow: SampleRow(format),
		NoteFa: entity.NoteFa,
	}
}
