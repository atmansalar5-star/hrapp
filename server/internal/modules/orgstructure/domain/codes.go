// کدهای نمودار سازمانی: شمارهٔ سمت، پایهٔ سمت سازمانی، پایهٔ شخصی و سه کد همراه.
//
// قالب و اعتبارسنجی در موتور قواعد است (`rules/orgcode.go`): «پیش‌شمارهٔ سه رقمی + نقطه +
// ده رقم» با شش پیش‌شمارهٔ متغیر ۰۸۰ تا ۰۸۵:
//
//	080 شمارهٔ سمت · 081 پایهٔ سمت سازمانی · 082 پایهٔ شخصی
//	083 شمارهٔ واحد سازمانی · 084 شمارهٔ انتصاب · 085 کد خانوادهٔ شغلی
//
// این فایل آن تعریف را به شناسنامهٔ رکورد «پست سازمانی» و دادهٔ نمونه وصل می‌کند تا هر جا
// پست ثبت یا ویرایش می‌شود، کد با همان طرح (پیش‌شماره‌های همان استقرار) سنجیده شود.
package domain

import (
	"strings"

	"hrms/internal/contracts"
	"hrms/internal/platform/recordkit"
	"hrms/internal/rules"
)

// کلید فیلدهای کد در شناسنامهٔ پست سازمانی.
const (
	FieldPostCode      = "postCode"
	FieldOrgGradeCode  = "orgGradeCode"
	FieldUnitCode      = "unitCode"
	FieldJobFamilyCode = "jobFamilyCode"
)

// positionCodeFields فیلدهای کد نمودار سازمانی برای نوع رکورد «پست سازمانی».
//
// عنوان فارسی و الگوی قالب از موتور قواعد خوانده می‌شود تا برچسب در دو جا نوشته نشود.
func positionCodeFields(scheme rules.OrgCodeScheme) []recordkit.Field {
	pattern := "." + strings.Repeat("x", rules.OrgCodeSerialDigits)
	code := func(key, kind string) recordkit.Field {
		return recordkit.Field{
			Key:      key,
			LabelFa:  rules.OrgCodeKindTitleFa(kind) + " (" + scheme.Prefix(kind) + pattern + ")",
			Type:     "text",
			Required: false,
		}
	}
	return []recordkit.Field{
		code(FieldPostCode, rules.OrgCodeKindPost),
		code(FieldOrgGradeCode, rules.OrgCodeKindOrgGrade),
		code(FieldUnitCode, rules.OrgCodeKindUnit),
		code(FieldJobFamilyCode, rules.OrgCodeKindJobFamily),
	}
}

// positionCodeFieldKinds نگاشت «کلید فیلد → نوع کد» برای اعتبارسنجی و کدگذاری.
func positionCodeFieldKinds() map[string]string {
	return map[string]string{
		FieldPostCode:      rules.OrgCodeKindPost,
		FieldOrgGradeCode:  rules.OrgCodeKindOrgGrade,
		FieldUnitCode:      rules.OrgCodeKindUnit,
		FieldJobFamilyCode: rules.OrgCodeKindJobFamily,
	}
}

// positionValidateWith قاعدهٔ کدهای پست سازمانی هنگام ایجاد/ویرایش.
//
// چرا اینجا: کد نمودار سازمانی شناسهٔ ماشین‌خوان است و اگر با قالب یا پیش‌شمارهٔ دیگری ذخیره
// شود، گزارش‌ها و انتصاب‌ها به آن پست نمی‌رسند. قاعده در دامنهٔ همین ماژول است و موتور رکورد
// فقط صدایش می‌زند.
func positionValidateWith(scheme rules.OrgCodeScheme) func(contracts.RecordInput) error {
	return func(in contracts.RecordInput) error {
		for field, kind := range positionCodeFieldKinds() {
			code := strings.TrimSpace(in.Fields[field])
			if code == "" {
				continue
			}
			if !scheme.ValidFor(code, kind) {
				return &recordkit.RuleError{Message: "«" + rules.OrgCodeKindTitleFa(kind) +
					"» باید با پیش‌شمارهٔ " + scheme.Prefix(kind) + " و قالب " +
					scheme.Prefix(kind) + "." + strings.Repeat("0", rules.OrgCodeSerialDigits) +
					" نوشته شود (نمونه: " + scheme.Sample(kind) + ")."}
			}
		}
		return nil
	}
}

// PostCodeOf شمارهٔ سمت یک رکورد پست؛ نبودِ کد ذخیره‌شده ⇒ ساخت کد از شناسهٔ رکورد.
func PostCodeOf(scheme rules.OrgCodeScheme, fields map[string]string, recordID string) string {
	return codeOf(scheme, fields, FieldPostCode, rules.OrgCodeKindPost, func() string {
		return scheme.ForRecord(rules.OrgCodeKindPost, recordID)
	})
}

// OrgGradeCodeOf پایهٔ سمت سازمانی؛ نبودِ کد ذخیره‌شده ⇒ ساخت کد از گروه شغلی پست.
func OrgGradeCodeOf(scheme rules.OrgCodeScheme, fields map[string]string, grade string) string {
	return codeOf(scheme, fields, FieldOrgGradeCode, rules.OrgCodeKindOrgGrade, func() string {
		if strings.TrimSpace(grade) == "" {
			grade = fields["grade"]
		}
		return scheme.OrgGradeFor(grade)
	})
}

// UnitCodeOf شمارهٔ واحد سازمانی؛ نبودِ کد ذخیره‌شده ⇒ ساخت کد از نام واحد.
func UnitCodeOf(scheme rules.OrgCodeScheme, fields map[string]string, recordID string) string {
	return codeOf(scheme, fields, FieldUnitCode, rules.OrgCodeKindUnit, func() string {
		if unit := strings.TrimSpace(fields["unit"]); unit != "" {
			return scheme.UnitFor(unit)
		}
		return scheme.ForRecord(rules.OrgCodeKindUnit, recordID)
	})
}

// JobFamilyCodeOf کد خانوادهٔ شغلی؛ نبودِ کد ذخیره‌شده ⇒ ساخت کد از خانوادهٔ شغلی پست.
func JobFamilyCodeOf(scheme rules.OrgCodeScheme, fields map[string]string) string {
	return codeOf(scheme, fields, FieldJobFamilyCode, rules.OrgCodeKindJobFamily, func() string {
		return scheme.JobFamilyFor(fields["jobFamily"])
	})
}

// codeOf کد ذخیره‌شده یا کد ساخته‌شده از دادهٔ پرونده.
func codeOf(scheme rules.OrgCodeScheme, fields map[string]string, field, kind string, derive func() string) string {
	if code := strings.TrimSpace(fields[field]); scheme.ValidFor(code, kind) {
		return code
	}
	if derive == nil {
		return ""
	}
	return derive()
}

// AssignmentCodeOf شمارهٔ انتصاب؛ نبودِ کد ذخیره‌شده ⇒ ساخت کد از شناسهٔ رکورد انتصاب.
func AssignmentCodeOf(scheme rules.OrgCodeScheme, fields map[string]string, recordID string) string {
	return codeOf(scheme, fields, "assignmentCode", rules.OrgCodeKindAssignment, func() string {
		return scheme.ForRecord(rules.OrgCodeKindAssignment, recordID)
	})
}

// PersonalGradeCodeOf کد پایهٔ شخصی از شمارهٔ پرسنلی.
func PersonalGradeCodeOf(scheme rules.OrgCodeScheme, employeeID string) string {
	return scheme.PersonalForEmployee(employeeID)
}

// applySeedCodes کدهای نمودار سازمانی را روی رکوردهای نمونهٔ پست‌ها می‌نشاند.
//
// شمارهٔ سمت به ترتیب رکورد (۰۸۰.۰۰۰۰۰۰۰۰۰۱ و ...) و پایهٔ سمت سازمانی از نردبان پایه‌ها
// ساخته می‌شود؛ کد واحد و خانوادهٔ شغلی هم از خودِ داده می‌آید تا دو پست هم‌واحد و هم‌خانواده
// یک کد بگیرند.
func applySeedCodes(scheme rules.OrgCodeScheme, items []contracts.RecordInput) {
	postSerial := int64(0)
	for i := range items {
		if items[i].Kind != KindPositions {
			continue
		}
		fields := items[i].Fields
		if fields == nil {
			fields = map[string]string{}
			items[i].Fields = fields
		}
		if strings.TrimSpace(fields[FieldPostCode]) == "" {
			postSerial++
			if code, ok := scheme.Format(rules.OrgCodeKindPost, postSerial); ok {
				fields[FieldPostCode] = code
			}
		}
		if strings.TrimSpace(fields[FieldOrgGradeCode]) == "" {
			if code := scheme.OrgGradeFor(fields["grade"]); code != "" {
				fields[FieldOrgGradeCode] = code
			}
		}
		if strings.TrimSpace(fields[FieldUnitCode]) == "" {
			if code := scheme.UnitFor(fields["unit"]); code != "" {
				fields[FieldUnitCode] = code
			}
		}
		if strings.TrimSpace(fields[FieldJobFamilyCode]) == "" {
			if code := scheme.JobFamilyFor(fields["jobFamily"]); code != "" {
				fields[FieldJobFamilyCode] = code
			}
		}
	}
}

// descriptorBase شناسهٔ پایهٔ ماژول (بدون دفتر فراتر) — در بردار عمومی بازنویسی می‌شود.
func descriptorWith(scheme rules.OrgCodeScheme) recordkit.Descriptor {
	out := descriptorBase()
	for i := range out.Kinds {
		if out.Kinds[i].Key != KindPositions {
			continue
		}
		out.Kinds[i].Fields = append(out.Kinds[i].Fields, positionCodeFields(scheme)...)
		out.Kinds[i].Validate = positionValidateWith(scheme)
	}
	return out
}

// Descriptor شناسهٔ ماژول ساختار سازمانی با پیش‌شماره‌های پیش‌فرض.
func Descriptor() recordkit.Descriptor {
	return descriptorWith(rules.DefaultOrgCodeScheme())
}

// DescriptorWith شناسهٔ ماژول با پیش‌شماره‌های همین استقرار.
func DescriptorWith(scheme rules.OrgCodeScheme) recordkit.Descriptor {
	return descriptorWith(scheme)
}

// Seed رکوردهای نمونهٔ سازمان با پیش‌شماره‌های پیش‌فرض.
func Seed() []contracts.RecordInput {
	return SeedWith(rules.DefaultOrgCodeScheme())
}

// SeedWith رکوردهای نمونهٔ سازمان با پیش‌شماره‌های همین استقرار.
func SeedWith(scheme rules.OrgCodeScheme) []contracts.RecordInput {
	out := seedBase()
	applySeedCodes(scheme, out)
	return out
}