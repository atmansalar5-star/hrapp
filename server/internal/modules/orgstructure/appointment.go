// اجرای ابلاغ انتصاب در تشکیلات — پیاده‌سازی پورت `contracts.AppointmentApplier`.
//
// چرا این فایل: «متصدی پست» در ساختار سازمانی از نوع رکورد `assignments` خوانده می‌شود،
// نه از فیلد پست در پروندهٔ کارمند. پس اجرای یک ابلاغ انتصاب/ترفیع تا وقتی انتصاب تازه
// در همین نوع رکورد ثبت نشود، در نمودار سازمانی دیده نمی‌شود و پست بلا تصدی می‌ماند.
//
// قاعدهٔ مالک داده اینجا می‌ماند: انتصاب تازه فقط با نتیجهٔ «احرازشده» و شواهد ثبت‌شده
// آغاز تصدی می‌شود (همان `appointmentRule` دامنه). ماژول انتصابات شواهد را از «دلیل و
// ملاک تصویب» خودِ ابلاغ می‌دهد؛ چون ابلاغی که زنجیرهٔ تصویب را گذرانده، احرازشده است.
package orgstructure

import (
	"context"
	"strconv"
	"strings"

	"hrms/internal/contracts"
	"hrms/internal/modules/orgstructure/domain"
	"hrms/internal/platform/modulekit"
	"hrms/internal/platform/recordkit"
)

// applier ثبت تصدی پست در تشکیلات از راه موتور رکورد همین ماژول.
type applier struct {
	svc *recordkit.Service
}

// registerApplier پورت «ثبت تصدی پست» را به مجموعهٔ پورت‌ها می‌دهد.
func registerApplier(deps modulekit.Deps, svc *recordkit.Service) {
	deps.Ports.SetAppointmentApplier(applier{svc: svc})
}

// Appoint یک انتصاب تازه ثبت می‌کند و انتصاب‌های پیشین همان کارمند را پایان می‌دهد.
//
// ترتیب عمدی است: نخست انتصاب تازه ثبت و آغاز تصدی می‌شود و تنها پس از موفقیت آن،
// انتصاب‌های پیشین بسته می‌شوند؛ اگر انتصاب تازه شکست بخورد، تصدی پیشین دست‌نخورده
// می‌ماند و سازمان بی‌متصدی نمی‌ماند.
func (a applier) Appoint(ctx context.Context, in contracts.PositionAppointment, actor string) (contracts.PositionAppointmentResult, error) {
	result := contracts.PositionAppointmentResult{}
	employeeID := strings.TrimSpace(in.EmployeeID)
	if employeeID == "" {
		return result, &recordkit.RuleError{Message: "اجرای ابلاغ بدون کارمند ممکن نیست."}
	}

	// کد پست: اگر ابلاغ کد نداشته باشد (رکورد قدیمی یا ثبت دستی)، پست از روی «عنوان»
	// در تشکیلات پیدا می‌شود؛ پس ابلاغ‌های موجود هم به پست خودشان می‌چسبند.
	code, positionTitle, matched := a.resolvePosition(ctx, strings.TrimSpace(in.PositionCode), strings.TrimSpace(in.PositionTitle))
	result.PositionMatched = matched
	if code == "" {
		return result, &recordkit.RuleError{Message: "کد پست سازمانی ابلاغ در تشکیلات پیدا نشد؛ " +
			"پست پیشنهادی را از فهرست پست‌های ساختار سازمانی انتخاب کنید."}
	}

	// انتصاب تازه: ابتدا به‌عنوان پیش‌نویس ثبت و سپس با اقدام «انتصاب و آغاز تصدی»
	// به وضعیت «در حال تصدی» می‌رود؛ درست همان مسیری که کاربر از فرم طی می‌کند.
	created, err := a.svc.Create(ctx, contracts.RecordInput{
		Kind:       domain.KindAssignments,
		Title:      assignmentTitle(positionTitle, code),
		EmployeeID: employeeID,
		Fields: map[string]string{
			"position":      code,
			"positionTitle": positionTitle,
			"unit":          strings.TrimSpace(in.Unit),
			"startDate":     strings.TrimSpace(in.StartDate),
			"decreeNo":      strings.TrimSpace(in.DecreeNo),
			"qualification": domain.QualificationMet,
			"evidence":      evidenceOf(in),
			"commitment":    numberText(in.Commitment),
		},
		NoteFa: "ثبت خودکار از اجرای ابلاغ انتصاب/ترفیع" + noteSuffix(in.NoteFa),
	}, actor)
	if err != nil {
		return result, err
	}
	if _, err := a.svc.Act(ctx, created.ID, contracts.RecordAction{Action: "appoint"}, actor); err != nil {
		return result, err
	}
	result.AssignmentID = created.ID

	// پایان تصدی‌های پیشین همین کارمند (جز همین انتصاب تازه).
	//
	// چرا با یادداشت: اقدام «ثبت پایان تصدی» دلیل می‌خواهد (قاعدهٔ دامنه)؛ بی آن، بستن
	// تصدی پیشین رد می‌شود و کارمند دو تصدی فعال هم‌زمان پیدا می‌کند. شکستِ هر بستن در
	// گزارش اجرا گفته می‌شود تا پنهان نماند.
	page, err := a.svc.List(ctx, contracts.RecordFilter{
		Kind: domain.KindAssignments, Status: domain.AssignmentActive,
		EmployeeID: employeeID, Scope: "all", Size: 200,
	})
	if err != nil {
		result.NoteFa = "انتصاب تازه ثبت شد، ولی خواندن انتصاب‌های پیشین برای پایان تصدی ممکن نشد."
		return result, nil
	}
	failed := 0
	for _, rec := range page.Items {
		if rec.ID == result.AssignmentID {
			continue
		}
		note := "پایان تصدی پیشین با اجرای ابلاغ " + strings.TrimSpace(in.DecreeNo)
		if strings.TrimSpace(in.DecreeNo) == "" {
			note = "پایان تصدی پیشین با اجرای ابلاغ انتصاب/ترفیع"
		}
		if _, err := a.svc.Act(ctx, rec.ID, contracts.RecordAction{Action: "end", NoteFa: note}, actor); err == nil {
			result.ClosedIDs = append(result.ClosedIDs, rec.ID)
		} else {
			failed++
		}
	}
	if failed > 0 {
		result.NoteFa = appointmentNoteFa(result) + " (" + strconv.Itoa(failed) +
			" انتصاب پیشین بسته نشد؛ در فهرست انتصاب‌های پست بررسی شود.)"
		return result, nil
	}
	result.NoteFa = appointmentNoteFa(result)
	return result, nil
}

// resolvePosition کد و عنوان پست را از تشکیلات پیدا می‌کند.
//
// نخست با کد (اگر ابلاغ کد داشته باشد) و در نبود آن با عنوان پست؛ مقایسهٔ عنوان پس از
// یکسان‌سازی فاصله و «ی/ک» انجام می‌شود تا «کارشناس ارشد مالي» و «کارشناس ارشد مالی»
// یک پست شناخته شوند. خروجی سوم می‌گوید پست در تشکیلات پیدا شد یا نه.
func (a applier) resolvePosition(ctx context.Context, code, title string) (string, string, bool) {
	page, err := a.svc.List(ctx, contracts.RecordFilter{
		Kind: domain.KindPositions, Scope: "all", Size: 500,
	})
	if err != nil {
		return "", "", false
	}
	wantCode := normalize(code)
	wantTitle := normalize(title)
	for _, rec := range page.Items {
		recordCode := strings.TrimSpace(rec.Fields["code"])
		if wantCode != "" && normalize(recordCode) == wantCode {
			return recordCode, firstOf(rec.Title, strings.TrimSpace(rec.Fields["positionTitle"]), title), true
		}
	}
	if wantTitle == "" {
		return "", "", false
	}
	for _, rec := range page.Items {
		if normalize(rec.Title) == wantTitle {
			return strings.TrimSpace(rec.Fields["code"]), rec.Title, true
		}
	}
	return "", "", false
}

// normalize متن را برای مقایسهٔ نام‌ها یکدست می‌کند.
func normalize(value string) string {
	replacer := strings.NewReplacer(
		"ي", "ی", "ك", "ک", "ۀ", "ه", "ة", "ه",
		"\u200c", "", " ", "", "\t", "",
	)
	return replacer.Replace(strings.TrimSpace(value))
}

// firstOf نخستین مقدار ناتهی.
func firstOf(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

// assignmentTitle عنوان انتصاب تازه؛ کد پست در نبود عنوان به کار می‌رود.
func assignmentTitle(title, code string) string {
	label := strings.TrimSpace(title)
	if label == "" {
		label = code
	}
	if label == "" {
		return "انتصاب سازمانی"
	}
	return "انتصاب " + label
}

// evidenceOf شواهد احراز انتصاب تازه از خود ابلاغ.
func evidenceOf(in contracts.PositionAppointment) string {
	parts := []string{}
	if reason := strings.TrimSpace(in.Evidence); reason != "" {
		parts = append(parts, reason)
	}
	if decree := strings.TrimSpace(in.DecreeNo); decree != "" {
		parts = append(parts, "ابلاغ "+decree)
	}
	if len(parts) == 0 {
		return "ابلاغ مصوب انتصابات، ترفیعات و کارراههٔ شغلی"
	}
	return strings.Join(parts, " — ")
}

// noteSuffix پسوند یادداشت ابلاغ.
func noteSuffix(note string) string {
	value := strings.TrimSpace(note)
	if value == "" {
		return ""
	}
	return " — " + value
}

// appointmentNoteFa متن گزارش نتیجهٔ ثبت تصدی.
func appointmentNoteFa(result contracts.PositionAppointmentResult) string {
	if !result.PositionMatched && result.AssignmentID != "" {
		return "تصدی ثبت شد؛ کد پست ابلاغ در تشکیلات پیدا نشد، پس پیوند پست در نمودار سازمانی برقرار نشد."
	}
	if len(result.ClosedIDs) > 0 {
		return "تصدی ثبت و انتصاب‌های پیشین پایان‌یافته شد."
	}
	return "تصدی پست در تشکیلات ثبت شد."
}

// numberText عدد صحیح برای فیلد متنی رکورد (خالی برای صفر و منفی).
func numberText(v int) string {
	if v <= 0 {
		return ""
	}
	return strconv.Itoa(v)
}
