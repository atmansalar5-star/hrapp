// قواعد وام کارکنان — پرداخت یک‌بارهٔ تسهیلات، جدول اقساط ماهانه، مانده بدهی و کد رسمی فیش.
//
// چرا در بستهٔ قواعد و نه در ماژول وام: صورت حساب وام دو مصرف‌کننده دارد — ماژول وام و
// بازنشستگی (صورت حساب، جدول اقساط، گزارش) و ماژول حقوق و مزایا (کسور فیش و بلوک اقساط).
// اگر هر کدام حساب خودش را می‌کرد، مبلغ قسط و مانده بدهی در دو جا دو عدد می‌شد. پس همهٔ
// حساب اینجاست و هر دو مصرف‌کننده از یک تعریف می‌خوانند.
//
// مدل مالی این سند:
//
//	جمع قابل بازپرداخت = اصل وام + کارمزد
//	مبلغ هر قسط       = جمع قابل بازپرداخت ÷ تعداد اقساط (باقیماندهٔ گردکردن در قسط آخر)
//	مانده بدهی        = جمع قابل بازپرداخت − اقساط کسرشده − پیش‌پرداخت − تسویه
//
// ماه‌ها شمسی‌اند و حساب ماه با «نمایهٔ ماه» انجام می‌شود (سال×۱۲ + ماه) تا گذر سال و
// دیرکرد بدون تبدیل تقویمی میلادی ممکن باشد.
package rules

import (
	"strconv"
	"strings"

	"hrms/internal/platform/calendar"
)

// انواع رخداد وام در تاریخچه و در پرداخت‌های ثبت‌شده.
const (
	// LoanEventDisburse پرداخت یک‌بارهٔ تسهیلات (بستانکار برای کارمند، ایجاد بدهی).
	LoanEventDisburse = "disburse"
	// LoanEventInstallment کسر قسط ماهانه از حقوق (کسور).
	LoanEventInstallment = "installment"
	// LoanEventPrepay پیش‌پرداخت (پرداخت نقدی برای کم‌کردن اصل بدهی).
	LoanEventPrepay = "prepay"
	// LoanEventSettle تسویهٔ کامل باقی‌ماندهٔ وام.
	LoanEventSettle = "settle"
)

// وضعیت هر ردیف جدول اقساط.
const (
	LoanRowPending = "pending"
	LoanRowPaid    = "paid"
	LoanRowOverdue = "overdue"
)

// LoanEventTitleFa عنوان فارسی نوع رخداد (برای تاریخچه و چاپ).
func LoanEventTitleFa(kind string) string {
	switch kind {
	case LoanEventDisburse:
		return "پرداخت یک‌بارهٔ تسهیلات"
	case LoanEventInstallment:
		return "کسر قسط از حقوق"
	case LoanEventPrepay:
		return "پیش‌پرداخت"
	case LoanEventSettle:
		return "تسویهٔ کامل"
	default:
		return kind
	}
}

// LoanRowStatusFa عنوان فارسی وضعیت قسط.
func LoanRowStatusFa(status string) string {
	switch status {
	case LoanRowPaid:
		return "کسرشده"
	case LoanRowOverdue:
		return "معوق"
	case LoanRowPending:
		return "در انتظار کسر"
	default:
		return status
	}
}

// LoanDirectionFa عنوان فارسی سوی جریان مالی (بستانکار = پرداخت به کارمند).
func LoanDirectionFa(kind string) string {
	switch kind {
	case LoanEventDisburse:
		return "بستانکار"
	default:
		return "کسور"
	}
}

// LoanTerms شرایط وام. مبلغ هر قسط و کارمزد اگر از بیرون نیاید، از همین قواعد ساخته
// می‌شود؛ پس فرم لازم نیست هیچ حسابی بکند.
type LoanTerms struct {
	// LoanID شناسهٔ رکورد وام (LOAN-0001).
	LoanID string `json:"loanId"`
	// LoanType کلید نوع وام دامنه (emergency، socialWork، housingDeposit ...).
	LoanType string `json:"loanType"`
	// Amount اصل وام (ریال).
	Amount int64 `json:"amount"`
	// FeeRatePercent نرخ کارمزد سالانه/کل بر پایهٔ مصوبهٔ همان نوع وام.
	FeeRatePercent float64 `json:"feeRatePercent"`
	// FeeAmount مبلغ کارمزد؛ صفر و نرخ مثبت ⇒ از نرخ محاسبه می‌شود.
	FeeAmount int64 `json:"feeAmount"`
	// Installments تعداد اقساط ماهانه.
	Installments int `json:"installments"`
	// InstallmentAmount مبلغ هر قسط (محاسبه‌شده؛ ورودی نیست).
	InstallmentAmount int64 `json:"installmentAmount"`
	// ReceivedDateFa تاریخ پرداخت یک‌بارهٔ تسهیلات (شمسی).
	ReceivedDateFa string `json:"receivedDateFa"`
	// FirstPeriodFa ماه شروع کسر اقساط (YYYY/MM)؛ خالی ⇒ ماه بعد از پرداخت.
	FirstPeriodFa string `json:"firstPeriodFa"`
}

// NormalizeLoanTerms شرایط وام را کامل و درست می‌کند: کارمزد از نرخ، مبلغ قسط از تعداد
// اقساط، و ماه شروع کسر از ماه بعدِ تاریخ پرداخت. خطاها فارسی و در سطح فیلد برمی‌گردند تا
// فرم همان کادر را نشان دهد.
func NormalizeLoanTerms(in LoanTerms) (LoanTerms, []string) {
	out := in
	problems := []string{}

	if out.Amount <= 0 {
		problems = append(problems, "مبلغ وام باید بزرگ‌تر از صفر باشد.")
	}
	if out.Installments <= 0 {
		problems = append(problems, "تعداد اقساط باید دست‌کم یک باشد.")
	}
	if out.Installments > 600 {
		problems = append(problems, "تعداد اقساط بیش از ۶۰۰ ماه پذیرفته نمی‌شود.")
	}
	if out.FeeRatePercent < 0 || out.FeeRatePercent > 100 {
		problems = append(problems, "نرخ کارمزد باید بین صفر و ۱۰۰ درصد باشد.")
	}
	if out.FeeAmount < 0 {
		problems = append(problems, "مبلغ کارمزد نمی‌تواند منفی باشد.")
	}
	if strings.TrimSpace(out.ReceivedDateFa) != "" {
		if _, err := calendar.Parse(out.ReceivedDateFa); err != nil {
			problems = append(problems, "تاریخ پرداخت وام باید به قالب شمسی YYYY/MM/DD باشد.")
		}
	}
	if strings.TrimSpace(out.FirstPeriodFa) != "" && allowanceMonth(out.FirstPeriodFa) == "" {
		problems = append(problems, "ماه شروع کسر اقساط باید به قالب شمسی YYYY/MM باشد.")
	}

	if out.FeeAmount == 0 && out.FeeRatePercent > 0 && out.Amount > 0 {
		out.FeeAmount = LoanFeeAmount(out.Amount, out.FeeRatePercent)
	}
	if out.FirstPeriodFa = allowanceMonth(out.FirstPeriodFa); out.FirstPeriodFa == "" {
		out.FirstPeriodFa = LoanFirstPeriod(out.ReceivedDateFa)
	}
	if out.Amount > 0 && out.Installments > 0 {
		out.InstallmentAmount = LoanInstallmentAmount(LoanTotalPayable(out.Amount, out.FeeAmount), out.Installments)
		if out.InstallmentAmount <= 0 {
			problems = append(problems, "مبلغ هر قسط صفر شد؛ مبلغ وام یا تعداد اقساط را بازبینی کنید.")
		}
	}
	return out, problems
}

// LoanPeriod ماه «YYYY/MM» را نرمال می‌کند (پذیرش ارقام فارسی و قالب تاریخ کامل) و برای
// ورودی نامعتبر رشتهٔ خالی می‌دهد. مصرف‌کننده: اعتبارسنجی دامنهٔ وام.
func LoanPeriod(raw string) string { return allowanceMonth(raw) }

// LoanTermsFromFields شرایط وام را از فیلدهای رکورد (کلید ماشین‌خوان ← مقدار متنی) می‌سازد.
//
// چرا در بستهٔ قواعد: دو مصرف‌کنندهٔ رکورد وام (ماژول وام برای صورت حساب و ماژول حقوق برای
// کسر قسط) هر دو همین رکورد را می‌بینند؛ نگاشت آن به ساختار قواعد فقط یک‌جا نوشته می‌شود تا
// مبلغ قسط و مانده بدهی در فیش و صورت حساب یکی باشد.
func LoanTermsFromFields(loanID string, f map[string]string) LoanTerms {
	get := func(key string) string { return strings.TrimSpace(f[key]) }
	return LoanTerms{
		LoanID:            loanID,
		LoanType:          get("loanType"),
		Amount:            int64Of(get("amount")),
		FeeRatePercent:    float64Of(get("feeRate")),
		FeeAmount:         int64Of(get("feeAmount")),
		Installments:      int(int64Of(get("installments"))),
		InstallmentAmount: int64Of(get("installmentAmount")),
		ReceivedDateFa:    get("receivedDate"),
		FirstPeriodFa:     get("firstPeriod"),
	}
}

// LoanRecordedPayment یک قسط ثبت‌شده در انبار رکوردها (خنثی از شکل رکورد، تا هر ماژول
// بتواند رکورد خودش را به آن نگاشت کند).
type LoanRecordedPayment struct {
	Seq        int
	PeriodFa   string
	PaidDateFa string
	Amount     int64
	MethodKey  string
	RefFa      string
	// Paid فقط اقساط «پرداخت‌شده/کسرشده» را می‌آورد؛ قسط در انتظار در پرداخت‌ها نمی‌آید.
	Paid bool
}

// LoanPaymentsFromRecords پرداخت‌های مؤثر یک وام را می‌سازد: اقساط کسرشده + مبلغ تسویهٔ
// ثبت‌شده روی خود رکورد وام.
func LoanPaymentsFromRecords(loanFields map[string]string, records []LoanRecordedPayment) []LoanPayment {
	out := make([]LoanPayment, 0, len(records)+1)
	for _, r := range records {
		if !r.Paid || r.Amount <= 0 {
			continue
		}
		out = append(out, LoanPayment{
			Seq: r.Seq, Kind: LoanEventInstallment, PeriodFa: LoanPeriod(r.PeriodFa),
			PaidDateFa: r.PaidDateFa, Amount: r.Amount,
			MethodFa: LoanPaymentMethodFa(r.MethodKey), CodeFa: "", RefFa: r.RefFa,
		})
	}
	if settled := int64Of(strings.TrimSpace(loanFields["settledAmount"])); settled > 0 {
		out = append(out, LoanPayment{
			Kind: LoanEventSettle, Amount: settled,
			PeriodFa:   LoanPeriod(loanFields["settleDate"]),
			PaidDateFa: strings.TrimSpace(loanFields["settleDate"]),
			MethodFa:   "تسویهٔ ثبت‌شده روی وام", RefFa: "settledAmount",
		})
	}
	return out
}

// LoanPaymentMethodFa عنوان فارسی روش پرداخت/کسر قسط.
func LoanPaymentMethodFa(key string) string {
	switch strings.TrimSpace(key) {
	case "salary":
		return "کسر از حقوق"
	case "cash":
		return "پرداخت نقدی"
	case "bank":
		return "واریز بانکی"
	case "":
		return ""
	default:
		return key
	}
}

// int64Of عدد صحیح را از رشته می‌خواند (ارقام فارسی، جداکنندهٔ هزارگان و اعشار).
func int64Of(raw string) int64 {
	value := normalizeAllowanceDigits(strings.TrimSpace(raw))
	value = strings.ReplaceAll(value, ",", "")
	value = strings.ReplaceAll(value, "،", "")
	if value == "" {
		return 0
	}
	if n, err := strconv.ParseInt(value, 10, 64); err == nil {
		return n
	}
	if f, err := strconv.ParseFloat(value, 64); err == nil {
		return int64(f)
	}
	return 0
}

// float64Of عدد اعشاری را از رشته می‌خواند (نرخ کارمزد).
func float64Of(raw string) float64 {
	value := normalizeAllowanceDigits(strings.TrimSpace(raw))
	value = strings.ReplaceAll(value, ",", "")
	value = strings.ReplaceAll(value, "٪", "")
	value = strings.TrimSuffix(value, "%")
	if value == "" {
		return 0
	}
	f, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return 0
	}
	return f
}

// LoanFeeAmount کارمزد وام = اصل × نرخ ÷ ۱۰۰ (گردکردن نیم‌به‌بالا).
func LoanFeeAmount(amount int64, ratePercent float64) int64 {
	if amount <= 0 || ratePercent <= 0 {
		return 0
	}
	return RoundHalfUp(float64(amount) * ratePercent / 100)
}

// LoanTotalPayable جمع قابل بازپرداخت = اصل وام + کارمزد.
func LoanTotalPayable(amount, fee int64) int64 {
	total := amount + fee
	if total < 0 {
		return 0
	}
	return total
}

// LoanInstallmentAmount مبلغ هر قسط: تقسیم گردشدهٔ جمع قابل بازپرداخت بر تعداد اقساط.
// باقیماندهٔ گردکردن در قسط آخر جبران می‌شود تا جمع اقساط دقیقاً برابر جمع قابل بازپرداخت
// باشد (در BuildLoanSchedule اعمال می‌شود).
func LoanInstallmentAmount(total int64, installments int) int64 {
	if total <= 0 || installments <= 0 {
		return 0
	}
	return RoundHalfUp(float64(total) / float64(installments))
}

// LoanFirstPeriod ماه شروع کسر اقساط در نبود مقدار صریح: ماه بعد از ماه پرداخت وام.
func LoanFirstPeriod(receivedDateFa string) string {
	period := allowanceMonth(receivedDateFa)
	if period == "" {
		return ""
	}
	return AddLoanMonths(period, 1)
}

// AddLoanMonths یک ماه شمسی «YYYY/MM» را به اندازهٔ n ماه جلو می‌برد (n می‌تواند منفی باشد).
// ماه‌شماری شمسی همیشه ۱۲ ماه دارد، پس حساب با «نمایهٔ ماه» انجام می‌شود.
func AddLoanMonths(periodFa string, n int) string {
	index, ok := loanMonthIndex(periodFa)
	if !ok {
		return ""
	}
	index += n
	if index < 0 {
		return ""
	}
	year := index / 12
	month := index%12 + 1
	return pad4(year) + "/" + pad2(month)
}

// LoanMonthsBetween فاصلهٔ ماه‌ها از a تا b (مثبت یعنی b بعد از a است).
func LoanMonthsBetween(a, b string) (int, bool) {
	ia, okA := loanMonthIndex(a)
	ib, okB := loanMonthIndex(b)
	if !okA || !okB {
		return 0, false
	}
	return ib - ia, true
}

// loanMonthIndex نمایهٔ ماه شمسی (سال×۱۲ + ماه−۱) برای مقایسه و جمع ماه‌ها.
func loanMonthIndex(periodFa string) (int, bool) {
	period := allowanceMonth(periodFa)
	if period == "" {
		return 0, false
	}
	year, errY := strconv.Atoi(period[:4])
	month, errM := strconv.Atoi(period[5:7])
	if errY != nil || errM != nil {
		return 0, false
	}
	return year*12 + month - 1, true
}

func pad4(v int) string {
	s := strconv.Itoa(v)
	for len(s) < 4 {
		s = "0" + s
	}
	return s
}

func pad2(v int) string {
	s := strconv.Itoa(v)
	if len(s) < 2 {
		s = "0" + s
	}
	return s
}

// LoanScheduleRow یک قسط از جدول اقساط.
type LoanScheduleRow struct {
	Seq        int    `json:"seq"`
	PeriodFa   string `json:"periodFa"`
	Amount     int64  `json:"amount"`
	Principal  int64  `json:"principal"`
	Fee        int64  `json:"fee"`
	Status     string `json:"status"`
	StatusFa   string `json:"statusFa"`
	PaidDateFa string `json:"paidDateFa,omitempty"`
	LoanID     string `json:"loanId,omitempty"`
}

// LoanSchedule جدول اقساط کامل یک وام.
type LoanSchedule struct {
	LoanID            string            `json:"loanId,omitempty"`
	Terms             LoanTerms         `json:"terms"`
	TotalPayable      int64             `json:"totalPayable"`
	InstallmentAmount int64             `json:"installmentAmount"`
	LastInstallment   int64             `json:"lastInstallment"`
	Rows              []LoanScheduleRow `json:"rows"`
	NoteFa            string            `json:"noteFa,omitempty"`
}

// BuildLoanSchedule جدول اقساط ماهانه را می‌سازد.
//
// قاعده‌های ثابت: هر قسط به اندازهٔ مبلغ قسط محاسبه‌شده است، قسط آخر باقیماندهٔ گردکردن را
// جبران می‌کند، و سهم اصل/کارمزد هر قسط به نسبت اصل به جمع قابل بازپرداخت تقسیم می‌شود؛
// سهم قسط آخر با تفاضل جمع‌ها تنظیم می‌شود تا هیچ ریالی گم یا تکرار نشود.
func BuildLoanSchedule(in LoanTerms) (LoanSchedule, []string) {
	terms, problems := NormalizeLoanTerms(in)
	schedule := LoanSchedule{LoanID: terms.LoanID, Terms: terms}
	if len(problems) > 0 {
		return schedule, problems
	}
	total := LoanTotalPayable(terms.Amount, terms.FeeAmount)
	schedule.TotalPayable = total
	schedule.InstallmentAmount = terms.InstallmentAmount

	rows := make([]LoanScheduleRow, 0, terms.Installments)
	var principalSum, feeSum int64
	for i := 0; i < terms.Installments; i++ {
		amount := terms.InstallmentAmount
		if i == terms.Installments-1 {
			amount = total - int64(terms.Installments-1)*terms.InstallmentAmount
			schedule.LastInstallment = amount
		}
		share := RoundHalfUp(float64(amount) * float64(terms.Amount) / float64(total))
		fee := amount - share
		if i == terms.Installments-1 {
			// قسط آخر: تفاضل تا جمع کل، تا گردکردن سهم‌ها هم جبران شود.
			share = terms.Amount - principalSum
			fee = terms.FeeAmount - feeSum
			amount = share + fee
		}
		principalSum += share
		feeSum += fee
		rows = append(rows, LoanScheduleRow{
			Seq:       i + 1,
			PeriodFa:  AddLoanMonths(terms.FirstPeriodFa, i),
			Amount:    amount,
			Principal: share,
			Fee:       fee,
			Status:    LoanRowPending,
			StatusFa:  LoanRowStatusFa(LoanRowPending),
			LoanID:    terms.LoanID,
		})
	}
	schedule.Rows = rows
	if terms.FirstPeriodFa == "" {
		// وام می‌تواند در مرحلهٔ «درخواست» بدون تاریخ پرداخت باشد؛ جدول ساخته می‌شود ولی
		// نبود سررسید صریح گزارش می‌گردد تا قسط بی‌ماه در فیش کسر نشود.
		problems = append(problems, "ماه شروع کسر اقساط تعیین نشده است (نه ماه شروع و نه تاریخ پرداخت وام).")
	}
	schedule.NoteFa = "مبلغ هر قسط " + formatRial(terms.InstallmentAmount) +
		" ریال و قسط آخر " + formatRial(schedule.LastInstallment) + " ریال است؛ " +
		"جمع اقساط دقیقاً برابر جمع قابل بازپرداخت (اصل + کارمزد) می‌شود."
	return schedule, problems
}

// LoanPayment یک پرداخت ثبت‌شده روی وام: قسط کسرشده از حقوق، پیش‌پرداخت یا تسویه.
type LoanPayment struct {
	// Seq شمارهٔ قسط؛ برای پیش‌پرداخت و تسویه صفر است.
	Seq        int    `json:"seq,omitempty"`
	Kind       string `json:"kind"`
	PeriodFa   string `json:"periodFa,omitempty"`
	PaidDateFa string `json:"paidDateFa,omitempty"`
	Amount     int64  `json:"amount"`
	MethodFa   string `json:"methodFa,omitempty"`
	CodeFa     string `json:"codeFa,omitempty"`
	RefFa      string `json:"refFa,omitempty"`
}

// LoanBalance مانده بدهی و وضعیت بازپرداخت یک وام.
type LoanBalance struct {
	TotalPayable          int64  `json:"totalPayable"`
	PaidAmount            int64  `json:"paidAmount"`
	PaidCount             int    `json:"paidCount"`
	OverdueCount          int    `json:"overdueCount"`
	OverdueAmount         int64  `json:"overdueAmount"`
	RemainingDebt         int64  `json:"remainingDebt"`
	RemainingInstallments int    `json:"remainingInstallments"`
	PrincipalRemaining    int64  `json:"principalRemaining"`
	FeeRemaining          int64  `json:"feeRemaining"`
	NextPeriodFa          string `json:"nextPeriodFa,omitempty"`
	NextAmount            int64  `json:"nextAmount,omitempty"`
	Settled               bool   `json:"settled"`
	NoteFa                string `json:"noteFa,omitempty"`
}

// LoanBalanceFor مانده بدهی را از جدول اقساط و پرداخت‌های ثبت‌شده می‌سازد.
//
// referencePeriodFa ماه مرجع گزارش است: قسطِ سررسیدشده و کسرنشده تا آن ماه «معوق» شمرده
// می‌شود. پرداخت قسط با شمارهٔ قسط تطبیق داده می‌شود (نه با ماه) تا اگر ماه کسر جابه‌جا شد،
// حساب خراب نشود.
func LoanBalanceFor(schedule LoanSchedule, payments []LoanPayment, referencePeriodFa string) LoanBalance {
	balance := LoanBalance{TotalPayable: schedule.TotalPayable}
	paidSeq := map[int]LoanPayment{}
	var extraPaid int64
	for _, p := range payments {
		if p.Amount <= 0 {
			continue
		}
		switch p.Kind {
		case LoanEventInstallment:
			if p.Seq > 0 {
				paidSeq[p.Seq] = p
			}
		case LoanEventPrepay, LoanEventSettle:
			extraPaid += p.Amount
		}
	}
	refIndex, hasRef := loanMonthIndex(referencePeriodFa)
	for _, row := range schedule.Rows {
		if p, ok := paidSeq[row.Seq]; ok {
			balance.PaidAmount += p.Amount
			balance.PaidCount++
			continue
		}
		if hasRef {
			if idx, ok := loanMonthIndex(row.PeriodFa); ok && idx < refIndex {
				balance.OverdueCount++
				balance.OverdueAmount += row.Amount
			}
		}
		balance.RemainingInstallments++
		if balance.NextPeriodFa == "" {
			balance.NextPeriodFa = row.PeriodFa
			balance.NextAmount = row.Amount
		}
	}
	balance.PaidAmount += extraPaid
	remaining := balance.TotalPayable - balance.PaidAmount
	if remaining < 0 {
		remaining = 0
	}
	balance.RemainingDebt = remaining
	if schedule.TotalPayable > 0 {
		balance.PrincipalRemaining = RoundHalfUp(float64(remaining) * float64(schedule.Terms.Amount) / float64(schedule.TotalPayable))
		if balance.PrincipalRemaining > remaining {
			balance.PrincipalRemaining = remaining
		}
	}
	balance.FeeRemaining = remaining - balance.PrincipalRemaining
	if balance.RemainingDebt == 0 {
		balance.Settled = true
		balance.NextPeriodFa = ""
		balance.NextAmount = 0
		balance.RemainingInstallments = 0
	}
	switch {
	case balance.Settled:
		balance.NoteFa = "وام تسویه شده است."
	case balance.OverdueCount > 0:
		balance.NoteFa = formatRial(balance.OverdueAmount) + " ریال از اقساط سررسید شده و کسر نشده است."
	default:
		balance.NoteFa = "قسط بعدی " + balance.NextPeriodFa + " به مبلغ " + formatRial(balance.NextAmount) + " ریال."
	}
	return balance
}

// LoanHistoryRow یک رخداد در صورت حساب وام.
type LoanHistoryRow struct {
	Order        int    `json:"order"`
	Kind         string `json:"kind"`
	KindFa       string `json:"kindFa"`
	TitleFa      string `json:"titleFa"`
	PeriodFa     string `json:"periodFa,omitempty"`
	DateFa       string `json:"dateFa,omitempty"`
	Seq          int    `json:"seq,omitempty"`
	CodeFa       string `json:"codeFa,omitempty"`
	Direction    string `json:"direction"`
	DirectionFa  string `json:"directionFa"`
	Amount       int64  `json:"amount"`
	StatusFa     string `json:"statusFa"`
	BalanceAfter int64  `json:"balanceAfter"`
	RefFa        string `json:"refFa,omitempty"`
}

// LoanStatement صورت حساب کامل یک وام: شرایط، جدول اقساط، تاریخچه و مانده بدهی.
type LoanStatement struct {
	LoanID                 string            `json:"loanId"`
	LoanType               string            `json:"loanType,omitempty"`
	LoanTypeFa             string            `json:"loanTypeFa,omitempty"`
	EmployeeID             string            `json:"employeeId,omitempty"`
	TitleFa                string            `json:"titleFa,omitempty"`
	StatusFa               string            `json:"statusFa,omitempty"`
	Amount                 int64             `json:"amount"`
	FeeRatePercent         float64           `json:"feeRatePercent"`
	FeeAmount              int64             `json:"feeAmount"`
	TotalPayable           int64             `json:"totalPayable"`
	Installments           int               `json:"installments"`
	InstallmentAmount      int64             `json:"installmentAmount"`
	LastInstallment        int64             `json:"lastInstallment"`
	FirstPeriodFa          string            `json:"firstPeriodFa,omitempty"`
	LastPeriodFa           string            `json:"lastPeriodFa,omitempty"`
	ReceivedDateFa         string            `json:"receivedDateFa,omitempty"`
	Disbursed              bool              `json:"disbursed"`
	PaymentCode            string            `json:"paymentCode,omitempty"`
	PaymentCodeTitleFa     string            `json:"paymentCodeTitleFa,omitempty"`
	InstallmentCode        string            `json:"installmentCode,omitempty"`
	InstallmentCodeTitleFa string            `json:"installmentCodeTitleFa,omitempty"`
	CodeNoteFa             string            `json:"codeNoteFa,omitempty"`
	Schedule               []LoanScheduleRow `json:"schedule"`
	History                []LoanHistoryRow  `json:"history"`
	Payments               []LoanPayment     `json:"payments"`
	Balance                LoanBalance       `json:"balance"`
	Problems               []string          `json:"problems,omitempty"`
	NoteFa                 string            `json:"noteFa,omitempty"`
}

// LoanStatementInput ورودی ساخت صورت حساب.
type LoanStatementInput struct {
	LoanID            string
	LoanType          string
	LoanTypeFa        string
	TitleFa           string
	EmployeeID        string
	StatusFa          string
	Terms             LoanTerms
	SettledAmount     int64
	SettleDateFa      string
	Payments          []LoanPayment
	ReferencePeriodFa string
}

// BuildLoanStatement صورت حساب را می‌سازد: از پرداخت یک‌بارهٔ تسهیلات تا اقساط و تسویه،
// با ماندهٔ بدهی پس از هر رخداد.
func BuildLoanStatement(in LoanStatementInput) (LoanStatement, []string) {
	schedule, problems := BuildLoanSchedule(in.Terms)
	statement := LoanStatement{
		LoanID:            in.LoanID,
		LoanType:          in.LoanType,
		LoanTypeFa:        in.LoanTypeFa,
		EmployeeID:        in.EmployeeID,
		TitleFa:           in.TitleFa,
		StatusFa:          in.StatusFa,
		Amount:            schedule.Terms.Amount,
		FeeRatePercent:    schedule.Terms.FeeRatePercent,
		FeeAmount:         schedule.Terms.FeeAmount,
		TotalPayable:      schedule.TotalPayable,
		Installments:      schedule.Terms.Installments,
		InstallmentAmount: schedule.InstallmentAmount,
		LastInstallment:   schedule.LastInstallment,
		FirstPeriodFa:     schedule.Terms.FirstPeriodFa,
		ReceivedDateFa:    schedule.Terms.ReceivedDateFa,
		Schedule:          schedule.Rows,
		Payments:          in.Payments,
		Problems:          problems,
	}
	if n := len(schedule.Rows); n > 0 {
		statement.LastPeriodFa = schedule.Rows[n-1].PeriodFa
	}

	// کد رسمی فیش همین نوع وام (پرداخت یک‌باره و کسر قسط).
	payment, installment, mapped := LoanPayslipCodeFor(in.LoanType)
	if mapped {
		statement.PaymentCode = payment.Code
		statement.PaymentCodeTitleFa = payment.TitleFa
	}
	if installment.Code != "" {
		statement.InstallmentCode = installment.Code
		statement.InstallmentCodeTitleFa = installment.TitleFa
	}
	statement.CodeNoteFa = LoanCodeNoteFa(in.LoanType)

	// پرداختهای مؤثر: پرداختهای قسط + پیشپرداخت/تسویهٔ ثبتشده روی خود رکورد وام.
	payments := append([]LoanPayment{}, in.Payments...)
	if in.SettledAmount > 0 && !hasPaymentKind(payments, LoanEventSettle) {
		payments = append(payments, LoanPayment{
			Kind: LoanEventSettle, Amount: in.SettledAmount,
			PeriodFa: allowanceMonth(in.SettleDateFa), PaidDateFa: in.SettleDateFa,
			RefFa: "مبلغ تسویهٔ ثبت‌شده روی رکورد وام",
		})
	}
	statement.Payments = payments
	balance := LoanBalanceFor(schedule, payments, in.ReferencePeriodFa)

	// وامی که تاریخ پرداخت ندارد هنوز پرداخت نشده است؛ پس بدهی واقعی صفر است و فقط
	// جدول اقساط (برنامهٔ آینده) نمایش داده می‌شود.
	disbursed := allowanceMonth(schedule.Terms.ReceivedDateFa) != ""
	statement.Disbursed = disbursed
	if !disbursed {
		balance.RemainingDebt = 0
		balance.PrincipalRemaining = 0
		balance.FeeRemaining = 0
		balance.Settled = false
		balance.NoteFa = "وام هنوز پرداخت نشده است؛ پس از ثبت تاریخ پرداخت، اقساط و مانده بدهی محاسبه می‌شود."
	}

	// ---------- تاریخچه ----------
	history := make([]LoanHistoryRow, 0, len(schedule.Rows)+2)
	debt := int64(0)
	order := 0
	add := func(row LoanHistoryRow) {
		order++
		row.Order = order
		row.DirectionFa = LoanDirectionFa(row.Kind)
		row.KindFa = LoanEventTitleFa(row.Kind)
		history = append(history, row)
	}
	// ۱) پرداخت یک‌بارهٔ تسهیلات: بدهی از همین‌جا ساخته می‌شود.
	if schedule.TotalPayable > 0 && disbursed {
		debt = schedule.TotalPayable
		add(LoanHistoryRow{
			Kind: LoanEventDisburse, TitleFa: "پرداخت وام به کارمند",
			DateFa: schedule.Terms.ReceivedDateFa, PeriodFa: allowanceMonth(schedule.Terms.ReceivedDateFa),
			CodeFa: statement.PaymentCode, Amount: schedule.Terms.Amount,
			StatusFa: "پرداخت‌شده", BalanceAfter: debt,
			RefFa: "اصل وام؛ کارمزد " + formatRial(schedule.Terms.FeeAmount) + " ریال به بدهی افزوده شد.",
		})
	}
	// ۲) پرداخت‌های ثبت‌شده: قسط کسرشده (با شمارهٔ قسط) جدا و پیش‌پرداخت/تسویه جدا نگه داشته
	// می‌شوند؛ ترتیب تاریخچه از پرداخت یک‌باره به اقساط و در پایان پرداخت‌های خارج از جدول
	// است تا ماندهٔ جاری پس از هر رخداد یکنوا (نزولی) بماند.
	paidSeq := map[int]LoanPayment{}
	extra := make([]LoanPayment, 0, 2)
	for _, p := range payments {
		if p.Amount <= 0 {
			continue
		}
		switch p.Kind {
		case LoanEventInstallment:
			if p.Seq > 0 {
				paidSeq[p.Seq] = p
			}
		case LoanEventPrepay, LoanEventSettle:
			extra = append(extra, p)
		}
	}
	// ۳) اقساط: کسرشده‌ها بدهی را کم می‌کنند، بقیه به‌عنوان برنامه نشان داده می‌شوند.
	for _, row := range schedule.Rows {
		statusFa := LoanRowStatusFa(row.Status)
		amount := row.Amount
		if p, ok := paidSeq[row.Seq]; ok {
			debt -= p.Amount
			if debt < 0 {
				debt = 0
			}
			statusFa = LoanRowStatusFa(LoanRowPaid)
			amount = p.Amount
			add(LoanHistoryRow{
				Kind: LoanEventInstallment, TitleFa: "قسط " + strconv.Itoa(row.Seq) + " وام",
				PeriodFa: row.PeriodFa, DateFa: p.PaidDateFa, Seq: row.Seq,
				CodeFa: statement.InstallmentCode, Amount: amount,
				StatusFa: statusFa, BalanceAfter: debt, RefFa: p.MethodFa,
			})
			continue
		}
		if referenceOverdue(row.PeriodFa, in.ReferencePeriodFa) {
			statusFa = LoanRowStatusFa(LoanRowOverdue)
		}
		// قسط کسرنشده بدهی را عوض نمی‌کند؛ فقط برنامه است.
		debtAfter := debt
		if balance.Settled {
			debtAfter = 0
		}
		add(LoanHistoryRow{
			Kind: LoanEventInstallment, TitleFa: "قسط " + strconv.Itoa(row.Seq) + " وام",
			PeriodFa: row.PeriodFa, Seq: row.Seq, CodeFa: statement.InstallmentCode,
			Amount: amount, StatusFa: statusFa, BalanceAfter: debtAfter,
		})
	}
	// ۴) پیش‌پرداخت و تسویهٔ ثبت‌شدهٔ روی وام (بیرون از جدول اقساط).
	for _, p := range extra {
		debt -= p.Amount
		if debt < 0 {
			debt = 0
		}
		add(LoanHistoryRow{
			Kind: p.Kind, TitleFa: LoanEventTitleFa(p.Kind), PeriodFa: p.PeriodFa, DateFa: p.PaidDateFa,
			Amount: p.Amount, CodeFa: statement.InstallmentCode,
			StatusFa: "ثبت‌شده", BalanceAfter: debt, RefFa: p.RefFa,
		})
	}
	statement.History = history
	statement.Balance = balance
	statement.NoteFa = schedule.NoteFa
	return statement, problems
}

// hasPaymentKind آیا پرداختی با این نوع ثبت شده است.
func hasPaymentKind(payments []LoanPayment, kind string) bool {
	for _, p := range payments {
		if p.Kind == kind && p.Amount > 0 {
			return true
		}
	}
	return false
}

// referenceOverdue آیا ماه قسط پیش از ماه مرجع است (سررسید گذشته و کسرنشده).
func referenceOverdue(periodFa, referencePeriodFa string) bool {
	idx, ok := loanMonthIndex(periodFa)
	if !ok {
		return false
	}
	ref, okRef := loanMonthIndex(referencePeriodFa)
	if !okRef {
		return false
	}
	return idx < ref
}

// LoanPayslipCodeFor کدهای رسمی فیش برای یک نوع وام: کد پرداخت یک‌بارهٔ تسهیلات و کد کسر
// قسط. کد نبودن یعنی آن نوع وام در جدول رسمی سازمان کد ندارد (صریح گزارش می‌شود).
func LoanPayslipCodeFor(loanType string) (payment PayslipCode, installment PayslipCode, ok bool) {
	switch strings.ToLower(strings.TrimSpace(loanType)) {
	case "emergency":
		payment, _ = PayslipCodeByNumber("500")
		installment, _ = PayslipCodeByNumber("5400")
	case "savings", "saving":
		payment, _ = PayslipCodeByNumber("502")
		installment, _ = PayslipCodeByNumber("5402")
	case "socialwork", "social_work":
		payment, _ = PayslipCodeByNumber("519")
	default:
		return PayslipCode{}, PayslipCode{}, false
	}
	return payment, installment, payment.Code != ""
}

// LoanCodeNoteFa توضیح فارسی وضعیت کد رسمی یک نوع وام.
func LoanCodeNoteFa(loanType string) string {
	payment, installment, ok := LoanPayslipCodeFor(loanType)
	if !ok {
		return "برای این نوع وام کد رسمی فیش ثبت نشده است؛ پرداخت و کسر آن با عنوان قلم گزارش می‌شود."
	}
	if installment.Code == "" {
		return "کد پرداخت " + payment.Code + " برای این نوع وام ثبت شده ولی کد کسر قسط در جدول رسمی نیست."
	}
	return "کد پرداخت " + payment.Code + " و کد کسر قسط " + installment.Code + "."
}

// loanTypeTitlesFa عنوان رسمی انواع وام کارکنان — منبع تنها برای عنوان، تا فیش و صورت
// حساب و فرم وام یک نام ببینند. نرخ کارمزد و سقف مصوب هر نوع وام «سیاست سازمان» است و در
// کاتالوگ ماژول وام می‌ماند.
var loanTypeTitlesFa = map[string]string{
	"housingDeposit":           "وام ودیعهٔ مسکن",
	"housingDepositSupplement": "متمم ودیعهٔ مسکن",
	"bank":                     "وام بانکی",
	"emergency":                "وام اضطراری",
	"socialWork":               "وام مددکاری",
	"firstDegreeDeath":         "وام فوت بستگان درجه یک",
	"unforeseen":               "وام حوادث مترقبه",
}

// LoanTypeTitleFa عنوان فارسی نوع وام؛ نوع ناشناخته همان کلید را برمی‌گرداند.
func LoanTypeTitleFa(key string) string {
	key = strings.TrimSpace(key)
	if title, ok := loanTypeTitlesFa[key]; ok {
		return title
	}
	return key
}

// LoanCodeGaps انواع وامی که کد رسمی فیش ندارند (کلید انواع از ماژول وام می‌آید).
func LoanCodeGaps(loanTypes []string) []string {
	out := []string{}
	for _, loanType := range loanTypes {
		if _, _, ok := LoanPayslipCodeFor(loanType); !ok {
			out = append(out, loanType)
		}
	}
	return out
}

// formatRial عدد را با جداکنندهٔ هزارگان برمی‌گرداند (برای پیام‌های فارسی).
func formatRial(v int64) string {
	sign := ""
	if v < 0 {
		sign = "-"
		v = -v
	}
	digits := strconv.FormatInt(v, 10)
	var out []byte
	for i, r := range digits {
		if i > 0 && (len(digits)-i)%3 == 0 {
			out = append(out, ',')
		}
		out = append(out, byte(r))
	}
	return sign + string(out)
}
