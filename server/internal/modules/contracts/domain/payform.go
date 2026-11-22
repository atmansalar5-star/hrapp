// زیرفرم‌های «حقوق و مزایا» بر پایهٔ نوع استخدام و نوع قرارداد.
//
// چرا در دامنه و نه در رابط: بخش حقوقی قرارداد برای هر نوع استخدام شکل دیگری دارد
// (جدول مزد رسمی برای رسمی/مدت معین، مبلغ و مبنای پیمان برای پیمانکاری، تفاهم‌نامهٔ
// پرداخت بین دو شرکت برای مأموریت). اگر این تفاوت در رابط نوشته شود، سرور عددی را
// می‌پذیرد که با آن نوع استخدام معنا ندارد و فیش حقوق بی‌سروصدا اشتباه می‌شود. پس:
//
//   - «شکل فرم» (کدام فیلدها، کدام لازم، کدام ممنوع) اینجا تعریف می‌شود؛
//   - سرور همان تعریف را به رابط می‌دهد (GET /api/v1/contracts/forms) و همان را در
//     زمان ثبت/ویرایش اعتبارسنجی می‌کند؛
//   - رابط فقط می‌کشد و می‌فرستد؛ هیچ قاعدهٔ کسب‌وکاری در UI نیست.
//
// سه خانوادهٔ پرداخت از نوع استخدام استخراج می‌شوند (نه از نوع قرارداد)، چون همین
// «نوع استخدام» است که در سامانه تعیین می‌کند کدام جدول مزد و کدام حکم اعمال شود.
package domain

import (
	"strconv"
	"strings"

	"hrms/internal/contracts"
	"hrms/internal/rules"
)

// خانواده‌های پرداخت.
const (
	// PayFamilyOfficial: رسمی، کارگر رسمی، مدت معین، قراردادی طویل‌المدت — جدول مزد رسمی.
	PayFamilyOfficial = "official"
	// PayFamilyContracting: پیمانکار ستادی/پروژه‌ای/جاری — مبلغ و مبنای پیمان.
	PayFamilyContracting = "contracting"
	// PayFamilySecondment: مأمور به/از سایر شرکت‌ها — تفاهم‌نامهٔ پرداخت بین دو شرکت.
	PayFamilySecondment = "secondment"
)

// PayFamilyFa عنوان فارسی هر خانواده (سرصفحهٔ زیرفرم در رابط).
var PayFamilyFa = map[string]string{
	PayFamilyOfficial:    "حقوق و مزایا — جدول مزد رسمی",
	PayFamilyContracting: "حقوق و مزایا — مبلغ و مبنای پیمان",
	PayFamilySecondment:  "حقوق و مزایا — تفاهم‌نامهٔ پرداخت (مأموریت)",
}

// PayFamilyNoteFa توضیح یک‌خطی هر خانواده در بالای زیرفرم.
var PayFamilyNoteFa = map[string]string{
	PayFamilyOfficial:    "اجزای ثابت مزد و فوق‌العاده‌های اقلیمی همین‌جا تعهد می‌شود؛ فیش حقوق و اسناد چاپی از حکم و در نبود آن از همین قرارداد می‌خوانند.",
	PayFamilyContracting: "در رابطهٔ پیمانکاری، مبنای پرداخت مبلغ و دورهٔ پیمان است و اجزای مزد رسمی (رتبه، سنوات، پست، اقلیمی) جای ندارد؛ اگر پیمانکار مزایایی می‌دهد، در یادداشت پیمان ثبت شود.",
	PayFamilySecondment:  "در مأموریت، روشن‌بودن «کدام شرکت حقوق می‌دهد» شرط اصلی است؛ اگر پرداخت تقسیم می‌شود، سهم شرکت مبدأ باید درصدی تعیین شود.",
}

// خانواده‌هایی که در آن‌ها جدول مزد رسمی نمایش داده می‌شود.

// نظر: کلیدهای فیلدهای پیمان/مأموریت در ستون JSONB قرارداد (pay_meta) ذخیره می‌شوند؛
// کلیدهای رسمی همان ستون‌های WageStructure هستند و در همان ستون‌های قبلی می‌مانند.

// PayFamilyOf خانوادهٔ پرداخت را از نوع استخدام برمی‌آورد.
//
// ورودی می‌تواند کلید لاتین یا عنوان فارسی باشد (همان قاعدهٔ EmploymentTypeByKey).
// نوع استخدام ناشناخته یعنی «رسمی» — محافظه‌کارانه‌ترین حالت، چون جدول مزد رسمی
// سخت‌گیرانه‌ترین اعتبارسنجی را دارد.
func PayFamilyOf(employmentType string) string {
	if t, ok := contracts.EmploymentTypeByKey(employmentType); ok {
		switch t.Kind {
		case contracts.EmploymentKindContracting:
			return PayFamilyContracting
		}
		switch t.Key {
		case "seconded_to", "seconded_from":
			return PayFamilySecondment
		}
		return PayFamilyOfficial
	}
	key := strings.ToLower(strings.TrimSpace(employmentType))
	switch {
	case strings.HasPrefix(key, "contractor"):
		return PayFamilyContracting
	case strings.HasPrefix(key, "seconded"):
		return PayFamilySecondment
	default:
		return PayFamilyOfficial
	}
}

// PayFamilyTitleFa عنوان فارسی خانواده؛ ورودی ناشناخته به «رسمی» می‌افتد.
func PayFamilyTitleFa(family string) string {
	if title, ok := PayFamilyFa[family]; ok {
		return title
	}
	return PayFamilyFa[PayFamilyOfficial]
}

// انواع فیلد در زیرفرم.
const (
	PayFieldMoney   = "money"
	PayFieldPercent = "percent"
	PayFieldDate    = "date"
	PayFieldText    = "text"
	PayFieldSelect  = "select"
)

// PayOption یک گزینهٔ فهرست انتخابی.
type PayOption struct {
	Value   string `json:"value"`
	LabelFa string `json:"labelFa"`
}

// PayCondition شرط وابسته به فیلد دیگر (لازم‌بودن یا ممنوعیت شرطی).
type PayCondition struct {
	// Field کلید فیلد دیگر در همان زیرفرم.
	Field string `json:"field"`
	// In مقادیری که با آن‌ها شرط برقرار است.
	In []string `json:"in,omitempty"`
	// Empty شرط «خالی‌بودن» فیلد دیگر.
	Empty bool `json:"empty,omitempty"`
	// MessageFa پیام خطا هنگام نقض شرط.
	MessageFa string `json:"messageFa,omitempty"`
}

// PayField یک فیلد زیرفرم.
type PayField struct {
	Key      string      `json:"key"`
	LabelFa  string      `json:"labelFa"`
	Type     string      `json:"type"`
	Required bool        `json:"required,omitempty"`
	UnitFa   string      `json:"unitFa,omitempty"`
	HintFa   string      `json:"hintFa,omitempty"`
	Options  []PayOption `json:"options,omitempty"`
	// DropOnSelect اگر پر شود، مقدار فیلد در زمان «تغییر نوع استخدام» دور ریخته می‌شود
	// (مثلاً دامنهٔ اقلیمی وقتی نوع استخدام از رسمی به پیمانکاری عوض می‌شود).
	DropOnSelect bool `json:"dropOnSelect,omitempty"`
	// RequiredWhen لازم است اگر شرط برقرار باشد.
	RequiredWhen *PayCondition `json:"requiredWhen,omitempty"`
	// ForbiddenWhen ممنوع است اگر شرط برقرار باشد.
	ForbiddenWhen *PayCondition `json:"forbiddenWhen,omitempty"`
}

// PayGroup گروه فیلدها (یک کارت در زیرفرم رابط).
type PayGroup struct {
	TitleFa string     `json:"titleFa"`
	HintFa  string     `json:"hintFa,omitempty"`
	Fields  []PayField `json:"fields"`
}

// PayForm مشخصات کامل زیرفرم یک خانوادهٔ پرداخت.
type PayForm struct {
	Family    string     `json:"family"`
	TitleFa   string     `json:"titleFa"`
	NoteFa    string     `json:"noteFa"`
	WageTable bool       `json:"wageTable"`
	Groups    []PayGroup `json:"groups"`
}

// ClimateRegionOptions گزینه‌های فهرست منطقهٔ اقلیمی (از جدول مصوب `rules`).
func ClimateRegionOptions() []PayOption {
	out := make([]PayOption, 0, len(rules.ClimateRegionsTable))
	for _, r := range rules.ClimateRegionsTable {
		out = append(out, PayOption{Value: r.Key, LabelFa: r.TitleFa})
	}
	return out
}

// PayFormOf زیرفرم خانوادهٔ پرداخت را می‌سازد.
//
// kind (نوع قرارداد: مدت دائم/موقت) در محتوای زیرفرم اثر ندارد — همان تفاوت با قاعدهٔ
// تاریخ پایان و تمدید در دامنه اعمال می‌شود — ولی نگه داشته می‌شود تا امضای تابع با
// انتخاب کاربر در فرم یکی باشد و بعداً بتوان قاعدهٔ مخصوص نوع قرارداد افزود.
func PayFormOf(employmentType, kind string) PayForm {
	_ = kind
	family := PayFamilyOf(employmentType)
	form := PayForm{
		Family:  family,
		TitleFa: PayFamilyTitleFa(family),
		NoteFa:  PayFamilyNoteFa[family],
		Groups:  []PayGroup{},
	}

	switch family {
	case PayFamilyContracting:
		form.Groups = []PayGroup{
			{
				TitleFa: "مشخصات پیمان",
				Fields: []PayField{
					{Key: "contractorName", LabelFa: "نام پیمانکار (شخص حقوقی)", Type: PayFieldText, Required: true},
					{Key: "contractNo", LabelFa: "شمارهٔ پیمان", Type: PayFieldText, Required: true},
					{Key: "contractSubject", LabelFa: "موضوع پیمان", Type: PayFieldText},
					{Key: "projectName", LabelFa: "پروژه/واحد محل خدمت", Type: PayFieldText},
				},
			},
			{
				TitleFa: "مبنای پرداخت",
				HintFa:  "مبلغ واردشده در «مبلغ پایهٔ ماهانه» بالا، مبلغ قرارداد این پیمان است و همان مبنای فیش/پرداخت می‌شود.",
				Fields: []PayField{
					{Key: "payBasis", LabelFa: "مبنای محاسبه", Type: PayFieldSelect, Required: true, Options: []PayOption{
						{Value: "monthly", LabelFa: "ماهانهٔ ثابت"},
						{Value: "daily", LabelFa: "روزمزد"},
						{Value: "hourly", LabelFa: "ساعت‌مزد"},
						{Value: "task", LabelFa: "کارمزدی / مقطوع"},
					}},
					{Key: "payAmount", LabelFa: "مبلغ مبنا", Type: PayFieldMoney, Required: true, UnitFa: "ریال",
						HintFa: "برای روزمزد/ساعت‌مزد: نرخ روز/ساعت؛ برای مقطوع: مبلغ کل."},
					{Key: "insurancePayer", LabelFa: "بیمهٔ تأمین اجتماعی", Type: PayFieldSelect, Required: true, Options: []PayOption{
						{Value: "contractor", LabelFa: "پیمانکار می‌پردازد"},
						{Value: "company", LabelFa: "شرکت (کارفرمای اصلی) می‌پردازد"},
						{Value: "none", LabelFa: "بدون بیمه (توافقی)"},
					}},
					{Key: "paymentSource", LabelFa: "مرجع پرداخت", Type: PayFieldSelect, Options: []PayOption{
						{Value: "company", LabelFa: "شرکت"},
						{Value: "contractor", LabelFa: "پیمانکار"},
					}},
					{Key: "invoiceNo", LabelFa: "شمارهٔ صورت‌وضعیت/فاکتور", Type: PayFieldText},
					{Key: "payNote", LabelFa: "یادداشت پرداخت", Type: PayFieldText},
				},
			},
		}
		// در پیمانکاری، اقلام رسمی ممنوع‌اند: نمایش داده نمی‌شوند و سرور هم رد می‌کند.
		form.WageTable = false

	case PayFamilySecondment:
		form.Groups = []PayGroup{
			{
				TitleFa: "طرفین مأموریت",
				Fields: []PayField{
					{Key: "hostCompany", LabelFa: "شرکت میزبان/مقصد", Type: PayFieldText, Required: true},
					{Key: "hostContact", LabelFa: "واحد تماس در شرکت مقصد", Type: PayFieldText},
				},
			},
			{
				TitleFa: "بازهٔ مأموریت",
				Fields: []PayField{
					{Key: "missionStart", LabelFa: "شروع مأموریت (شمسی)", Type: PayFieldDate, Required: true},
					{Key: "missionEnd", LabelFa: "پایان مأموریت (شمسی)", Type: PayFieldDate},
				},
			},
			{
				TitleFa: "ترتیب پرداخت",
				Fields: []PayField{
					{Key: "salaryPayer", LabelFa: "پرداخت حقوق", Type: PayFieldSelect, Required: true, Options: []PayOption{
						{Value: "origin", LabelFa: "شرکت مبدأ (همین سازمان)"},
						{Value: "host", LabelFa: "شرکت مقصد"},
						{Value: "shared", LabelFa: "تقسیم بین دو شرکت"},
					}},
					{Key: "originSharePercent", LabelFa: "سهم شرکت مبدأ", Type: PayFieldPercent, UnitFa: "درصد",
						RequiredWhen: &PayCondition{Field: "salaryPayer", In: []string{"host", "shared"},
							MessageFa: "وقتی پرداخت با شرکت مقصد یا تقسیمی است، سهم شرکت مبدأ باید تعیین شود."}},
					{Key: "allowancesPayer", LabelFa: "پرداخت فوق‌العاده‌ها (اقلیمی/مأموریت)", Type: PayFieldSelect, Options: []PayOption{
						{Value: "origin", LabelFa: "شرکت مبدأ"},
						{Value: "host", LabelFa: "شرکت مقصد"},
					}},
					{Key: "missionNote", LabelFa: "شمارهٔ تفاهم‌نامه/یادداشت", Type: PayFieldText},
				},
			},
		}
		// اجزای مزد رسمی در سند مأموریت ثبت نمی‌شود: مزد کارمند از حکم/پارامتر شرکت
		// می‌آید و این سند فقط «ترتیب پرداخت بین دو شرکت» را تعیین می‌کند.
		form.WageTable = false

	default: // PayFamilyOfficial
		form.WageTable = true
		form.Groups = []PayGroup{
			{
				TitleFa: "منطقهٔ اقلیمی محل خدمت",
				Fields: []PayField{
					{Key: "climateRegion", LabelFa: "منطقهٔ اقلیمی", Type: PayFieldSelect, Options: ClimateRegionOptions(),
						HintFa: "انتخاب منطقه، ضرایب پیشنهادی را می‌آورد؛ اگر ضرایب دستی پر شوند، همان ملاک است."},
				},
			},
			{
				TitleFa: "اجزای ثابت مزد (ریال در ماه)",
				Fields: []PayField{
					{Key: "rankWage", LabelFa: "مزد رتبه", Type: PayFieldMoney, UnitFa: "ریال"},
					{Key: "seniorityWage", LabelFa: "مزد سنوات", Type: PayFieldMoney, UnitFa: "ریال"},
					{Key: "postAllowance", LabelFa: "فوق‌العادهٔ پست", Type: PayFieldMoney, UnitFa: "ریال"},
					{Key: "sacrificeAllowance", LabelFa: "فوق‌العادهٔ ایثارگری", Type: PayFieldMoney, UnitFa: "ریال"},
					{Key: "climateAllowance", LabelFa: "فوق‌العادهٔ اقلیمی مصوب", Type: PayFieldMoney, UnitFa: "ریال",
						HintFa: "مبلغ ثابت و جزو مزد مبنا؛ در برابر ضرایب درصدی که روی مزد مبنا سوار می‌شوند."},
					{Key: "salaryRepair", LabelFa: "ترمیم حقوق", Type: PayFieldMoney, UnitFa: "ریال"},
				},
			},
			{
				TitleFa: "تفاوت تطبیق",
				Fields: []PayField{
					{Key: "payGap", LabelFa: "تفاوت تطبیق ماهانه", Type: PayFieldMoney, UnitFa: "ریال",
						HintFa: "جذب‌شدنی است، جزو مزد مبنا نیست و مشمول بیمه و مالیات است."},
					{Key: "payGapSince", LabelFa: "تاریخ شروع تفاوت تطبیق (شمسی)", Type: PayFieldDate},
				},
			},
			{
				TitleFa: "ضرایب اقلیمی همین کارمند (درصد)",
				HintFa:  "خالی بماند یعنی ضریب پیشنهادی منطقه/پارامتر شرکت اعمال می‌شود.",
				Fields: []PayField{
					{Key: "badClimatePercent", LabelFa: "بدی آب و هوا", Type: PayFieldPercent, UnitFa: "درصد"},
					{Key: "deprivationPercent", LabelFa: "محرومیت از تسهیلات زندگی", Type: PayFieldPercent, UnitFa: "درصد"},
					{Key: "regionHardshipPercent", LabelFa: "صعوبت شرایط منطقه", Type: PayFieldPercent, UnitFa: "درصد"},
					{Key: "extraTimePercent", LabelFa: "جبران صرف وقت اضافی", Type: PayFieldPercent, UnitFa: "درصد"},
				},
			},
		}
	}

	// خانوادهٔ پرداخت: گروه «دستهٔ پرداخت» در همهٔ خانواده‌ها هست، چون روش پرداخت و گروه
	// جدول ۳ به شکل زیرفرم حقوق وابسته نیستند.
	if form.Groups == nil {
		form.Groups = []PayGroup{}
	}
	form.Groups = append([]PayGroup{payClassificationGroup(form.Family)}, form.Groups...)

	return form
}

// payClassificationGroup کارت «دستهٔ پرداخت» را می‌سازد: روش پرداخت و گروه جدول ۳.
func payClassificationGroup(family string) PayGroup {
	note := "روش پرداخت می‌گوید حکم و فیش با کدام موتور حساب شوند"
	if family == PayFamilyOfficial {
		note += "؛ کارمند رسمی نفت همیشه گروه یکم و روش «نظام پرداخت صندوق بازنشستگی نفت» است."
	} else {
		note += "؛ خانواده‌های پیمان‌کاری و مأموریت با «حقوق و دستمزد جاری» محاسبه می‌شوند."
	}
	groups := make([]PayOption, 0, len(PayGroupOptions()))
	for _, group := range PayGroupOptions() {
		groups = append(groups, PayOption{Value: strconv.Itoa(group.Number),
			LabelFa: group.TitleFa})
	}
	models := make([]PayOption, 0, len(PayModelOptions()))
	for _, model := range PayModelOptions() {
		label := model.TitleFa
		if model.Optional {
			label += " (اختیاری)"
		}
		models = append(models, PayOption{Value: model.Key, LabelFa: label})
	}
	return PayGroup{
		TitleFa: "دستهٔ پرداخت (روش پرداخت و گروه جدول ۳)",
		HintFa:  note,
		Fields: []PayField{
			{Key: "payModel", LabelFa: "روش پرداخت", Type: PayFieldSelect, Options: models,
				HintFa: "از چهار روش مصوب سامانه: حقوق و دستمزد جاری، صندوق نفت، ۳۴۰۰۶، ۴پی."},
			{Key: "payGroup", LabelFa: "گروه نظام پرداخت نفت (جدول ۳)", Type: PayFieldSelect, Options: groups,
				HintFa: "درصد حق جذب عمومی و حق مقام از همین گروه می‌آید؛ گروه یکم مخصوص کارمند رسمی نفت است."},
		},
	}
}

// PayFormsOf همهٔ زیرفرم‌های شناخته‌شده (برای یک بار گرفتن رابط).
func PayFormsOf() []PayForm {
	families := []string{PayFamilyOfficial, PayFamilyContracting, PayFamilySecondment}
	out := make([]PayForm, 0, len(families))
	for _, f := range families {
		// خانواده را مستقیم می‌سازیم تا استقلال از نوع استخدام نمونه حفظ شود.
		form := formOfFamily(f)
		out = append(out, form)
	}
	return out
}

// formOfFamily زیرفرم یک خانواده را بدون واسطهٔ نوع استخدام می‌سازد.
func formOfFamily(family string) PayForm {
	switch family {
	case PayFamilyContracting:
		return PayFormOf("contractor_staff", KindFixedTerm)
	case PayFamilySecondment:
		return PayFormOf("seconded_to", KindFixedTerm)
	default:
		return PayFormOf("official_oil", KindPermanent)
	}
}

// PayInput دادهٔ ورودی اعتبارسنجی زیرفرم.
type PayInput struct {
	Kind           string
	EmploymentType string
	MonthlySalary  int64
	Wage           contracts.WageStructure
	Meta           map[string]any
	// PayModel روش پرداخت انتخابی و PayGroup گروه جدول ۳؛ صفر/خالی یعنی خودکار.
	PayModel string
	PayGroup int
}

// PayFieldError خطای یک فیلد زیرفرم (فارسی، آمادهٔ نمایش کنار همان فیلد).
type PayFieldError struct {
	Field   string
	Message string
}

// PayFamilyOfInput خانوادهٔ ورودی را برمی‌آورد (نوع استخدام، و در نبود آن نوع قرارداد).
func PayFamilyOfInput(in PayInput) string {
	if strings.TrimSpace(in.EmploymentType) != "" {
		return PayFamilyOf(in.EmploymentType)
	}
	// بدون نوع استخدام: قرارداد مدت موقت ⇒ پیمانکاری نیست، ولی محافظه‌کارانه رسمی.
	return PayFamilyOfficial
}

// ValidatePayForm ورودی را با تعریف زیرفرم همان خانواده می‌سنجد.
//
// این تابع تنها مرجع اعتبارسنجی بخش «حقوق و مزایا» است: رابط همان تعریف را می‌کشد و
// سرور همین را اجرا می‌کند، پس UI و API هرگز واگرا نمی‌شوند.
func ValidatePayForm(in PayInput) []PayFieldError {
	family := PayFamilyOfInput(in)
	form := PayFormOf(in.EmploymentType, in.Kind)
	var errs []PayFieldError

	values := payValues(family, in)

	for _, group := range form.Groups {
		for _, field := range group.Fields {
			raw := values[field.Key]
			empty := isEmptyValue(raw)
			if field.Required && empty {
				errs = append(errs, PayFieldError{Field: field.Key, Message: field.LabelFa + " الزامی است."})
				continue
			}
			if field.RequiredWhen != nil && conditionHolds(*field.RequiredWhen, values) && empty {
				message := field.RequiredWhen.MessageFa
				if message == "" {
					message = field.LabelFa + " الزامی است."
				}
				errs = append(errs, PayFieldError{Field: field.Key, Message: message})
				continue
			}
			if field.ForbiddenWhen != nil && conditionHolds(*field.ForbiddenWhen, values) && !empty {
				message := field.ForbiddenWhen.MessageFa
				if message == "" {
					message = field.LabelFa + " در این حالت مجاز نیست."
				}
				errs = append(errs, PayFieldError{Field: field.Key, Message: message})
				continue
			}
			if empty {
				continue
			}
			if field.Type == PayFieldSelect && len(field.Options) > 0 && !optionAllowed(field.Options, asString(raw)) {
				errs = append(errs, PayFieldError{Field: field.Key, Message: field.LabelFa + " مقدار مجاز ندارد."})
			}
			if field.Type == PayFieldPercent {
				if p, ok := asFloat(raw); ok && (p < 0 || p > 1) {
					errs = append(errs, PayFieldError{Field: field.Key, Message: field.LabelFa + " باید درصدی بین ۰ و ۱۰۰ باشد."})
				}
			}
			if field.Type == PayFieldMoney {
				if m, ok := asFloat(raw); ok && m < 0 {
					errs = append(errs, PayFieldError{Field: field.Key, Message: field.LabelFa + " نمی‌تواند منفی باشد."})
				}
			}
		}
	}

	// قاعدهٔ خانوادگی: در خانواده‌های غیررسمی هیچ جزء مزد رسمی ثبت نمی‌شود، حتی اگر
	// فراخوان مستقیم API آن را بفرستد (رابط آن فیلدها را نشان نمی‌دهد).
	if family != PayFamilyOfficial {
		if in.Wage.HasWageComponents() || in.Wage.HasClimatePercent() || strings.TrimSpace(in.Wage.ClimateRegion) != "" {
			message := "در قرارداد پیمانکاری اجزای مزد رسمی (رتبه، سنوات، پست، اقلیمی، ترمیم، تفاوت تطبیق) و منطقهٔ اقلیمی جای ندارد؛ مبنای پرداخت همان مبلغ پیمان است."
			if family == PayFamilySecondment {
				message = "در سند مأموریت اجزای مزد رسمی و منطقهٔ اقلیمی ثبت نمی‌شود؛ مزد از حکم کارگزینی می‌آید و این سند فقط ترتیب پرداخت بین دو شرکت را تعیین می‌کند."
			}
			errs = append(errs, PayFieldError{Field: "wageStructure", Message: message})
		}
		if in.MonthlySalary <= 0 {
			errs = append(errs, PayFieldError{Field: "monthlySalary", Message: "مبلغ پایهٔ ماهانه الزامی است."})
		}
	}

	if in.MonthlySalary <= 0 && family == PayFamilyOfficial {
		errs = append(errs, PayFieldError{Field: "monthlySalary", Message: "مبلغ پایهٔ ماهانه الزامی است."})
	}

	// دستهٔ پرداخت: روش پرداخت و گروه جدول ۳ باید با نوع استخدام سازگار باشند.
	errs = append(errs, ValidatePayClassification(in.EmploymentType, in.PayGroup, in.PayModel)...)

	return errs
}

// payValues نگاشت یکدست «کلید فیلد → مقدار» از دو منبع می‌سازد: اجزای مزد (ستون‌های
// ثابت) و فرادادهٔ خانواده (JSONB). کلید تکراری از فراداده مقدم است.
func payValues(family string, in PayInput) map[string]any {
	values := map[string]any{
		"climateRegion":         in.Wage.ClimateRegion,
		"rankWage":              in.Wage.RankWage,
		"seniorityWage":         in.Wage.SeniorityWage,
		"postAllowance":         in.Wage.PostAllowance,
		"sacrificeAllowance":    in.Wage.SacrificeAllowance,
		"climateAllowance":      in.Wage.ClimateAllowance,
		"salaryRepair":          in.Wage.SalaryRepair,
		"payGap":                in.Wage.PayGap,
		"payGapSince":           in.Wage.PayGapSince,
		"badClimatePercent":     in.Wage.BadClimatePercent,
		"deprivationPercent":    in.Wage.DeprivationPercent,
		"regionHardshipPercent": in.Wage.RegionHardshipPercent,
		"extraTimePercent":      in.Wage.ExtraTimePercent,
		"monthlySalary":         in.MonthlySalary,
	}
	for key, value := range in.Meta {
		values[key] = value
	}
	return values
}

func conditionHolds(condition PayCondition, values map[string]any) bool {
	raw := values[condition.Field]
	if condition.Empty {
		return isEmptyValue(raw)
	}
	current := asString(raw)
	for _, want := range condition.In {
		if current == want {
			return true
		}
	}
	return false
}

func isEmptyValue(value any) bool {
	switch v := value.(type) {
	case nil:
		return true
	case string:
		return strings.TrimSpace(v) == ""
	case float64:
		return v == 0
	case int64:
		return v == 0
	case int:
		return v == 0
	case bool:
		return false // بولی false یک مقدار معتبر است (مثلاً «بیمه ندارد»)
	default:
		return false
	}
}

func optionAllowed(options []PayOption, value string) bool {
	for _, o := range options {
		if o.Value == value {
			return true
		}
	}
	return false
}

func asString(value any) string {
	if s, ok := value.(string); ok {
		return strings.TrimSpace(s)
	}
	return ""
}

func asFloat(value any) (float64, bool) {
	switch v := value.(type) {
	case float64:
		return v, true
	case int64:
		return float64(v), true
	case int:
		return float64(v), true
	default:
		return 0, false
	}
}
