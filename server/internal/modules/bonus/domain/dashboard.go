// منطق دامنهٔ ماژول «bonus» — «dashboard»: قواعد، اعتبارسنجی و محاسبات خالص بدون وابستگی به زیرساخت.

package domain

import (
	"sort"
	"strings"

	"hrms/internal/contracts"
)

// PaymentFact یک رویداد پرداخت پاداش، مستقل از انبار و لایهٔ HTTP.
//
// داشبورد روی همین «واقعیت‌ها» ساخته می‌شود تا منطق تجمیع در دامنه بماند و آزمون‌پذیر
// باشد؛ لایهٔ api فقط رکوردهای انبار را به این ساختار تبدیل می‌کند.
type PaymentFact struct {
	EmployeeID   string
	EmployeeName string
	Management   string
	BonusType    string
	Amount       int64
	Date         string
	Period       string
	Status       string
	// Group گروه کارگزینی کارمند (شرکتی / ارکان ثالث) از کاتالوگ انواع استخدام.
	Group string
}

// BonusStatuses فهرست وضعیت‌های گردش‌کار پرداخت پاداش به ترتیب اعلامی.
func BonusStatuses() []contracts.RecordStatus {
	kind := Descriptor().Kinds[0]
	out := make([]contracts.RecordStatus, 0, len(kind.Statuses))
	for _, s := range kind.Statuses {
		out = append(out, contracts.RecordStatus{Value: s.Value, LabelFa: s.LabelFa})
	}
	return out
}

// BuildDashboard داشبورد مدیریتی پاداش را از رویدادهای پرداخت می‌سازد.
//
// قاعده‌ها: پرداخت لغوشده در هیچ جمعی نمی‌آید (فقط شمارش می‌شود)، هر ردیف مدیریت حتی
// بدون پرداخت حاضر است تا نمودار خلأ نداشته باشد، و «تعداد دفعات پرداخت» شمارش رویدادها
// است نه شمارش نفرات.
func BuildDashboard(facts []PaymentFact) contracts.BonusDashboard {
	out := contracts.BonusDashboard{
		ManagementsTotal: len(contracts.Managements()),
		NoteFa:           "هر ردیف یک رویداد پرداخت است؛ «دفعات» شمارش همین رویدادهاست. پرداخت‌های لغوشده در جمع‌ها نمی‌آیند و فقط در برش وضعیت شمرده می‌شوند.",
	}

	type mgmtAgg struct {
		payments      int
		beneficiaries map[string]bool
		total         float64
		max           float64
		company       int
		third         int
		first, last   string
	}
	mgmt := map[string]*mgmtAgg{}
	for _, m := range contracts.Managements() {
		mgmt[m.Key] = &mgmtAgg{beneficiaries: map[string]bool{}}
	}

	type empKey struct{ emp, mgmt string }
	emp := map[empKey]*contracts.BonusEmployeeRow{}
	empTypes := map[empKey]map[string]bool{}

	typeAgg := map[string]*contracts.BonusTypeRow{}
	periodAgg := map[string]*contracts.BonusPeriodRow{}
	periodPeople := map[string]map[string]bool{}
	statusAgg := map[string]*contracts.BonusStatusRow{}
	for _, s := range BonusStatuses() {
		statusAgg[s.Value] = &contracts.BonusStatusRow{StatusKey: s.Value, StatusFa: s.LabelFa}
	}

	beneficiaries := map[string]bool{}
	var paidAmount float64

	for _, f := range facts {
		st := statusAgg[f.Status]
		if st == nil {
			st = &contracts.BonusStatusRow{StatusKey: f.Status, StatusFa: f.Status}
			statusAgg[f.Status] = st
		}
		st.Payments++
		st.Amount += float64(f.Amount)

		if f.Status == "cancelled" {
			out.CancelledPayments++
			continue
		}
		amount := float64(f.Amount)
		out.TotalPayments++
		out.TotalAmount += amount
		beneficiaries[f.EmployeeID] = true
		if f.Status == "paid" {
			out.PaidPayments++
			paidAmount += amount
		} else if f.Status == "approved" {
			out.ApprovedPayments++
		} else {
			out.DraftPayments++
		}
		if amount > out.MaxPayment {
			out.MaxPayment = amount
		}
		if out.MinPayment == 0 || amount < out.MinPayment {
			out.MinPayment = amount
		}
		switch f.Group {
		case EmploymentGroupThirdParty:
			out.ThirdPartyAmount += amount
		default:
			out.CompanyAmount += amount
		}

		if a := mgmt[f.Management]; a != nil {
			a.payments++
			a.beneficiaries[f.EmployeeID] = true
			a.total += amount
			if amount > a.max {
				a.max = amount
			}
			if f.Group == EmploymentGroupThirdParty {
				a.third++
			} else {
				a.company++
			}
			a.first, a.last = minDate(a.first, f.Date), maxDate(a.last, f.Date)
		}

		k := empKey{f.EmployeeID, f.Management}
		row := emp[k]
		if row == nil {
			row = &contracts.BonusEmployeeRow{
				EmployeeID: f.EmployeeID, EmployeeName: f.EmployeeName,
				ManagementKey: f.Management, ManagementFa: contracts.ManagementTitleFa(f.Management),
			}
			emp[k] = row
			empTypes[k] = map[string]bool{}
		}
		row.Payments++
		row.TotalAmount += amount
		row.FirstDate, row.LastDate = minDate(row.FirstDate, f.Date), maxDate(row.LastDate, f.Date)
		empTypes[k][BonusTypeTitleFa(f.BonusType)] = true

		if tr := typeAgg[f.BonusType]; tr == nil {
			typeAgg[f.BonusType] = &contracts.BonusTypeRow{
				TypeKey: f.BonusType, TypeFa: BonusTypeTitleFa(f.BonusType), Payments: 1, Amount: amount,
			}
		} else {
			tr.Payments++
			tr.Amount += amount
		}

		if f.Period != "" {
			pr := periodAgg[f.Period]
			if pr == nil {
				pr = &contracts.BonusPeriodRow{Period: f.Period}
				periodAgg[f.Period] = pr
				periodPeople[f.Period] = map[string]bool{}
			}
			pr.Payments++
			pr.Amount += amount
			periodPeople[f.Period][f.EmployeeID] = true
		}
	}

	covered := 0
	out.ManagementRows = make([]contracts.BonusManagementRow, 0, len(contracts.Managements()))
	for _, m := range contracts.Managements() {
		a := mgmt[m.Key]
		row := contracts.BonusManagementRow{
			ManagementKey: m.Key, ManagementFa: m.TitleFa,
			Payments: a.payments, Beneficiaries: len(a.beneficiaries),
			TotalAmount: a.total, MaxPayment: a.max,
			CompanyCount: a.company, ThirdPartyCount: a.third,
			FirstDate: a.first, LastDate: a.last,
		}
		if row.Beneficiaries > 0 {
			row.AveragePerPerson = round2(a.total / float64(row.Beneficiaries))
			covered++
		}
		out.ManagementRows = append(out.ManagementRows, row)
	}
	out.ManagementsCovered = covered

	out.EmployeeRows = make([]contracts.BonusEmployeeRow, 0, len(emp))
	for k, row := range emp {
		row.AveragePayment = round2(row.TotalAmount / float64(row.Payments))
		titles := make([]string, 0, len(empTypes[k]))
		for t := range empTypes[k] {
			titles = append(titles, t)
		}
		sort.Strings(titles)
		row.TypeTitlesFa = titles
		out.EmployeeRows = append(out.EmployeeRows, *row)
	}
	sort.Slice(out.EmployeeRows, func(i, j int) bool {
		if out.EmployeeRows[i].TotalAmount != out.EmployeeRows[j].TotalAmount {
			return out.EmployeeRows[i].TotalAmount > out.EmployeeRows[j].TotalAmount
		}
		return out.EmployeeRows[i].EmployeeID < out.EmployeeRows[j].EmployeeID
	})

	out.TypeRows = make([]contracts.BonusTypeRow, 0, len(bonusTypes()))
	for _, t := range bonusTypes() {
		row := contracts.BonusTypeRow{TypeKey: t.Key, TypeFa: t.TitleFa}
		if tr := typeAgg[t.Key]; tr != nil {
			row.Payments, row.Amount = tr.Payments, tr.Amount
		}
		row.SharePercent = percent(row.Amount, out.TotalAmount)
		out.TypeRows = append(out.TypeRows, row)
	}
	// نوع‌های ناشناختهٔ دادهٔ قدیمی گم نشوند.
	for key, tr := range typeAgg {
		if _, ok := BonusTypeByKey(key); ok {
			continue
		}
		tr.SharePercent = percent(tr.Amount, out.TotalAmount)
		out.TypeRows = append(out.TypeRows, *tr)
	}

	periods := make([]string, 0, len(periodAgg))
	for p := range periodAgg {
		periods = append(periods, p)
	}
	sort.Strings(periods)
	out.PeriodRows = make([]contracts.BonusPeriodRow, 0, len(periods))
	for _, p := range periods {
		row := *periodAgg[p]
		row.Beneficiaries = len(periodPeople[p])
		out.PeriodRows = append(out.PeriodRows, row)
	}

	out.StatusRows = make([]contracts.BonusStatusRow, 0, len(statusAgg))
	for _, s := range BonusStatuses() {
		out.StatusRows = append(out.StatusRows, *statusAgg[s.Value])
	}

	out.TotalBeneficiaries = len(beneficiaries)
	out.PaidAmount = paidAmount
	if out.TotalPayments > 0 {
		out.AveragePerPayment = round2(out.TotalAmount / float64(out.TotalPayments))
	}
	if out.TotalBeneficiaries > 0 {
		out.AveragePerPerson = round2(out.TotalAmount / float64(out.TotalBeneficiaries))
	}
	return out
}

// گروه‌های کارگزینی — همان تفکیک جدول سهمیهٔ تسهیلات، برای برش «شرکتی / ارکان ثالث».
const (
	EmploymentGroupCompany    = "company"
	EmploymentGroupThirdParty = "thirdParty"
)

// minDate و maxDate مقایسهٔ تاریخ شمسی به‌صورت رشته‌ای (قالب ثابت YYYY/MM/DD این را
// ممکن می‌کند) و نگه‌داشتن مقدار موجود وقتی تاریخ تازه خالی است.
func minDate(a, b string) string {
	b = strings.TrimSpace(b)
	if b == "" {
		return a
	}
	if a == "" || b < a {
		return b
	}
	return a
}

func maxDate(a, b string) string {
	b = strings.TrimSpace(b)
	if b == "" {
		return a
	}
	if b > a {
		return b
	}
	return a
}

// percent نسبت را به درصد با یک رقم اعشار تبدیل می‌کند؛ مخرج صفر یعنی صفر.
func percent(part, total float64) float64 {
	if total <= 0 {
		return 0
	}
	return float64(int(part/total*1000+0.5)) / 10
}

// round2 گرد کردن به دو رقم اعشار (برای میانگین‌های ریالی).
func round2(v float64) float64 { return float64(int(v*100+0.5)) / 100 }
