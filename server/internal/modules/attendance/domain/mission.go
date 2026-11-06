package domain

import "strings"

// این فایل فهرست‌های مرجع «مأموریت» را نگه می‌دارد: نوع مأموریت (اداری، اعزام پزشکی، خارج از
// کشور)، مشمول هزینهٔ اعزام پزشکی (بیمار یا همراه بیمار)، نوع اسکان، ارز پرداخت و نوع سند
// پشتیبان (قبض تاکسی و فاکتورهای مورد تأیید). کلیدها ماشین‌خوان و برچسب‌ها فارسی‌اند تا فرم،
// اعتبارسنجی و موتور محاسبه از یک منبع بخوانند.

// MissionOption یک گزینهٔ انتخابی (کلید ماشین‌خوان + عنوان فارسی).
type MissionOption struct {
	Key     string `json:"key"`
	TitleFa string `json:"titleFa"`
	NoteFa  string `json:"noteFa,omitempty"`
}

// missionTypeCatalog انواع مأموریت.
var missionTypeCatalog = []MissionOption{
	{Key: "administrative", TitleFa: "مأموریت اداری",
		NoteFa: "سفر کاری داخل کشور؛ فوقالعادهٔ روزانهٔ ریالی، اقامت و ایاب‌وذهاب با اسناد پشتیبان."},
	{Key: "medical", TitleFa: "اعزام پزشکی",
		NoteFa: "اعزام درمانی کارمند یا همراه بیمار؛ اسکان و هزینه بر پایهٔ مشمول (بیمار/همراه بیمار)."},
	{Key: "foreign", TitleFa: "مأموریت خارج از کشور",
		NoteFa: "پرداخت دوگانهٔ ریالی و ارزی؛ نرخ تسعیر و فوقالعادهٔ ارزی در پارامترهای شرکت تنظیم می‌شود."},
}

// medicalBeneficiaryCatalog مشمول هزینه در اعزام پزشکی.
var medicalBeneficiaryCatalog = []MissionOption{
	{Key: "patient", TitleFa: "بیمار",
		NoteFa: "هزینهٔ اسکان، ایاب‌وذهاب و وعده‌ها برای خود بیمار پرداخت می‌شود."},
	{Key: "companion", TitleFa: "همراه بیمار",
		NoteFa: "هزینهٔ اسکان و ایاب‌وذهاب همراه پرداخت می‌شود؛ سهم وعده‌ها بر پایهٔ پارامتر شرکت است."},
}

// accommodationCatalog نوع اسکان در مأموریت (اداری و اعزام پزشکی).
var accommodationCatalog = []MissionOption{
	{Key: "", TitleFa: "بدون اسکان سازمانی"},
	{Key: "hotel", TitleFa: "هتل"},
	{Key: "guesthouse", TitleFa: "مهمانسرا"},
	{Key: "hospital", TitleFa: "بیمارستان / محل درمان"},
	{Key: "relative", TitleFa: "منزل اقوام"},
	{Key: "dormitory", TitleFa: "خوابگاه سازمانی"},
}

// currencyCatalog ارزهای پرداخت مأموریت (ریالی و ارزی).
var currencyCatalog = []MissionOption{
	{Key: "IRR", TitleFa: "ریال ایران"},
	{Key: "EUR", TitleFa: "یورو"},
	{Key: "USD", TitleFa: "دلار آمریکا"},
	{Key: "AED", TitleFa: "درهم امارات"},
	{Key: "GBP", TitleFa: "پوند انگلیس"},
}

// documentTypeCatalog انواع سند پشتیبان مأموریت: قبض تاکسی و فاکتورهای مورد تأیید.
var documentTypeCatalog = []MissionOption{
	{Key: "taxi", TitleFa: "قبض تاکسی",
		NoteFa: "قبض تاکسی شهری یا تاکسی فرودگاه؛ پس از تأیید در جمع ایاب‌وذهاب می‌آید."},
	{Key: "hotel", TitleFa: "فاکتور هتل"},
	{Key: "guesthouse", TitleFa: "فاکتور مهمانسرا"},
	{Key: "air_ticket", TitleFa: "بلیت هواپیما"},
	{Key: "bus_ticket", TitleFa: "بلیت اتوبوس"},
	{Key: "mileage", TitleFa: "قبض کرایهٔ کیلومتری"},
	{Key: "medical", TitleFa: "فاکتور درمانی"},
	{Key: "other", TitleFa: "سایر فاکتورهای تأییدشده"},
}

// MissionTypes فهرست انواع مأموریت.
func MissionTypes() []MissionOption { return append([]MissionOption{}, missionTypeCatalog...) }

// MedicalBeneficiaries فهرست مشمول‌های هزینهٔ اعزام پزشکی.
func MedicalBeneficiaries() []MissionOption {
	return append([]MissionOption{}, medicalBeneficiaryCatalog...)
}

// Accommodations فهرست انواع اسکان.
func Accommodations() []MissionOption { return append([]MissionOption{}, accommodationCatalog...) }

// MissionCurrencies فهرست ارزهای پرداخت.
func MissionCurrencies() []MissionOption { return append([]MissionOption{}, currencyCatalog...) }

// MissionDocumentTypes فهرست انواع سند پشتیبان.
func MissionDocumentTypes() []MissionOption { return append([]MissionOption{}, documentTypeCatalog...) }

// optionTitleFa عنوان فارسی یک گزینه؛ کلید ناشناخته خودش برمی‌گردد.
func optionTitleFa(list []MissionOption, key string) string {
	value := strings.TrimSpace(key)
	for _, item := range list {
		if item.Key == value {
			return item.TitleFa
		}
	}
	return value
}

// MissionTypeTitleFa عنوان فارسی نوع مأموریت.
func MissionTypeTitleFa(key string) string { return optionTitleFa(missionTypeCatalog, key) }

// MedicalBeneficiaryTitleFa عنوان فارسی مشمول هزینهٔ اعزام پزشکی.
func MedicalBeneficiaryTitleFa(key string) string {
	return optionTitleFa(medicalBeneficiaryCatalog, key)
}

// AccommodationTitleFa عنوان فارسی نوع اسکان.
func AccommodationTitleFa(key string) string { return optionTitleFa(accommodationCatalog, key) }

// CurrencyTitleFa عنوان فارسی ارز.
func CurrencyTitleFa(key string) string {
	return optionTitleFa(currencyCatalog, strings.ToUpper(strings.TrimSpace(key)))
}

// DocumentTypeTitleFa عنوان فارسی نوع سند پشتیبان.
func DocumentTypeTitleFa(key string) string { return optionTitleFa(documentTypeCatalog, key) }

// ValidMissionType آیا نوع مأموریت مجاز است.
func ValidMissionType(key string) bool { return validOption(missionTypeCatalog, key) }

// ValidMedicalBeneficiary آیا مشمول هزینهٔ اعزام پزشکی مجاز است.
func ValidMedicalBeneficiary(key string) bool { return validOption(medicalBeneficiaryCatalog, key) }

// ValidCurrency آیا ارز پرداخت مجاز است.
func ValidCurrency(key string) bool {
	value := strings.ToUpper(strings.TrimSpace(key))
	for _, item := range currencyCatalog {
		if item.Key == value {
			return true
		}
	}
	return false
}

// ValidDocumentType آیا نوع سند پشتیبان مجاز است.
func ValidDocumentType(key string) bool { return validOption(documentTypeCatalog, key) }

// validOption سنجش کلید در یک فهرست گزینه.
func validOption(list []MissionOption, key string) bool {
	value := strings.TrimSpace(key)
	for _, item := range list {
		if item.Key == value {
			return true
		}
	}
	return false
}
