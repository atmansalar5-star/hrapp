// آزمون‌های قواعد وام: شرایط، جدول اقساط ماهانه، مانده بدهی و تاریخچه، و کد رسمی فیش.
package rules

import (
	"strings"
	"testing"
)

// TestLOAN01LoanTerms شرایط وام را می‌سنجد: کارمزد از نرخ، مبلغ قسط از تعداد اقساط،
// ماه شروع کسر از ماه بعدِ پرداخت، و رد ورودی نامعتبر.
func TestLOAN01LoanTerms(t *testing.T) {
	terms, problems := NormalizeLoanTerms(LoanTerms{
		LoanID: "LOAN-0001", LoanType: "housingDeposit",
		Amount: 800_000_000, FeeRatePercent: 4, Installments: 60, ReceivedDateFa: "1404/03/15",
	})
	if len(problems) != 0 {
		t.Fatalf("شرایط معتبر نباید ایراد داشته باشد: %v", problems)
	}
	if terms.FeeAmount != 32_000_000 {
		t.Fatalf("کارمزد ۴٪ از ۸۰۰ میلیون باید ۳۲ میلیون باشد، %d شد", terms.FeeAmount)
	}
	total := LoanTotalPayable(terms.Amount, terms.FeeAmount)
	if total != 832_000_000 {
		t.Fatalf("جمع قابل بازپرداخت باید ۸۳۲ میلیون باشد، %d شد", total)
	}
	if terms.InstallmentAmount != 13_866_667 {
		t.Fatalf("مبلغ هر قسط باید ۱۳٬۸۶۶٬۶۶۷ باشد، %d شد", terms.InstallmentAmount)
	}
	if terms.FirstPeriodFa != "1404/04" {
		t.Fatalf("ماه شروع کسر باید ماه بعد از پرداخت (1404/04) باشد، %q شد", terms.FirstPeriodFa)
	}
	if LoanFeeAmount(100_000_000, 18) != 18_000_000 {
		t.Fatal("کارمزد ۱۸٪ از ۱۰۰ میلیون باید ۱۸ میلیون باشد")
	}
	if LoanInstallmentAmount(0, 6) != 0 || LoanInstallmentAmount(60_000_000, 0) != 0 {
		t.Fatal("تقسیم با ورودی صفر باید صفر بدهد")
	}

	bad := []LoanTerms{
		{Amount: 0, Installments: 12},
		{Amount: 10_000_000, Installments: 0},
		{Amount: 10_000_000, Installments: 12, FeeRatePercent: 140},
		{Amount: 10_000_000, Installments: 12, FeeRatePercent: -2},
		{Amount: 10_000_000, Installments: 12, FirstPeriodFa: "1405/13"},
		{Amount: 10_000_000, Installments: 12, ReceivedDateFa: "1405/01"},
		{Amount: 10_000_000, Installments: 900},
	}
	for i, in := range bad {
		if _, problems := NormalizeLoanTerms(in); len(problems) == 0 {
			t.Fatalf("ورودی نامعتبر شمارهٔ %d باید ایراد بدهد: %+v", i, in)
		}
	}
}

// TestLOAN02Schedule جدول اقساط را می‌سنجد: ماه‌شماری شمسی، پذیرش ارقام فارسی، جبران
// باقیماندهٔ گردکردن در قسط آخر و تقسیم اصل/کارمزد روی اقساط.
func TestLOAN02Schedule(t *testing.T) {
	if got := AddLoanMonths("1404/12", 1); got != "1405/01" {
		t.Fatalf("گذر سال: انتظار 1405/01، %q شد", got)
	}
	if got := AddLoanMonths("1405/01", -1); got != "1404/12" {
		t.Fatalf("عقب‌گرد سال: انتظار 1404/12، %q شد", got)
	}
	if got := AddLoanMonths("1405/01", 12); got != "1406/01" {
		t.Fatalf("یک سال جلو: انتظار 1406/01، %q شد", got)
	}
	if got := LoanFirstPeriod("۱۴۰۴/۰۱/۱۰"); got != "1404/02" {
		t.Fatalf("ارقام فارسی پذیرفته نشد: %q شد", got)
	}

	schedule, problems := BuildLoanSchedule(LoanTerms{
		LoanID: "LOAN-0001", Amount: 800_000_000, FeeRatePercent: 4, Installments: 60,
		ReceivedDateFa: "1404/03/15",
	})
	if len(problems) != 0 {
		t.Fatalf("جدول معتبر نباید ایراد داشته باشد: %v", problems)
	}
	if len(schedule.Rows) != 60 {
		t.Fatalf("جدول باید ۶۰ قسط داشته باشد، %d شد", len(schedule.Rows))
	}
	if schedule.Rows[0].PeriodFa != "1404/04" || schedule.Rows[59].PeriodFa != "1409/03" {
		t.Fatalf("ماه‌های ابتدا/انتها درست نیست: %q تا %q", schedule.Rows[0].PeriodFa, schedule.Rows[59].PeriodFa)
	}
	if schedule.Rows[0].Amount != 13_866_667 {
		t.Fatalf("مبلغ قسط اول باید ۱۳٬۸۶۶٬۶۶۷ باشد، %d شد", schedule.Rows[0].Amount)
	}
	if schedule.LastInstallment != 13_866_647 {
		t.Fatalf("قسط آخر باید باقیماندهٔ گردکردن را جبران کند (۱۳٬۸۶۶٬۶۴۷)، %d شد", schedule.LastInstallment)
	}
	var amounts, principals, fees int64
	for _, row := range schedule.Rows {
		amounts += row.Amount
		principals += row.Principal
		fees += row.Fee
		if row.Amount != row.Principal+row.Fee {
			t.Fatalf("قسط %d: مبلغ با جمع اصل و کارمزد نمی‌خواند: %+v", row.Seq, row)
		}
	}
	if amounts != schedule.TotalPayable {
		t.Fatalf("جمع اقساط (%d) باید برابر جمع قابل بازپرداخت (%d) باشد", amounts, schedule.TotalPayable)
	}
	if principals != 800_000_000 || fees != 32_000_000 {
		t.Fatalf("جمع اصل/کارمزد اقساط باید ۸۰۰/۳۲ میلیون باشد: %d / %d", principals, fees)
	}

	// وام بی‌تاریخ: جدول ساخته می‌شود ولی نبود سررسید صریح گزارش می‌گردد.
	noDate, problems := BuildLoanSchedule(LoanTerms{Amount: 30_000_000, Installments: 3})
	if len(noDate.Rows) != 3 {
		t.Fatalf("جدول بی‌تاریخ باید ۳ قسط بسازد، %d ساخت", len(noDate.Rows))
	}
	if len(problems) == 0 || !strings.Contains(strings.Join(problems, " "), "ماه شروع") {
		t.Fatalf("نبود ماه شروع باید صریح گزارش شود: %v", problems)
	}

	// ورودی نامعتبر باید جدول نسازد.
	empty, problems := BuildLoanSchedule(LoanTerms{Amount: 10_000_000, Installments: 0})
	if len(empty.Rows) != 0 || len(problems) == 0 {
		t.Fatal("تعداد اقساط صفر باید بی‌جدول و با ایراد برگردد")
	}

	if n, ok := LoanMonthsBetween("1404/04", "1409/03"); !ok || n != 59 {
		t.Fatalf("فاصلهٔ ماه‌ها باید ۵۹ باشد، %d شد", n)
	}
}

// TestLOAN03Balance مانده بدهی را می‌سنجد: اقساط کسرشده، معوق، پیش‌پرداخت و تسویه.
func TestLOAN03Balance(t *testing.T) {
	schedule, problems := BuildLoanSchedule(LoanTerms{
		Amount: 30_000_000, Installments: 3, FirstPeriodFa: "1405/01",
	})
	if len(problems) != 0 {
		t.Fatalf("جدول نباید ایراد داشته باشد: %v", problems)
	}

	// هیچ قسطی کسر نشده و ماه مرجع ۱۴۰۵/۰۳ است: دو قسط اول معوق‌اند.
	none := LoanBalanceFor(schedule, nil, "1405/03")
	if none.RemainingDebt != 30_000_000 || none.RemainingInstallments != 3 {
		t.Fatalf("مانده بدهی اولیه باید ۳۰ میلیون و ۳ قسط باشد: %+v", none)
	}
	if none.OverdueCount != 2 || none.OverdueAmount != 20_000_000 {
		t.Fatalf("اقساط معوق تا ۱۴۰۵/۰۳ باید ۲ قسط/۲۰ میلیون باشد: %+v", none)
	}
	if none.NextPeriodFa != "1405/01" || none.NextAmount != 10_000_000 {
		t.Fatalf("قسط بعدی باید ۱۴۰۵/۰۱ به مبلغ ۱۰ میلیون باشد: %+v", none)
	}

	// قسط اول کسر شده است.
	paidOne := LoanBalanceFor(schedule, []LoanPayment{
		{Seq: 1, Kind: LoanEventInstallment, Amount: 10_000_000, PaidDateFa: "1405/01/20"},
	}, "1405/02")
	if paidOne.PaidAmount != 10_000_000 || paidOne.PaidCount != 1 {
		t.Fatalf("جمع کسرشده باید ۱۰ میلیون و یک قسط باشد: %+v", paidOne)
	}
	if paidOne.RemainingDebt != 20_000_000 || paidOne.RemainingInstallments != 2 {
		t.Fatalf("مانده بدهی پس از یک قسط باید ۲۰ میلیون و ۲ قسط باشد: %+v", paidOne)
	}
	if paidOne.NextPeriodFa != "1405/02" || paidOne.OverdueCount != 0 {
		t.Fatalf("قسط بعدی باید ۱۴۰۵/۰۲ و بدون معوقه باشد: %+v", paidOne)
	}

	// پیش‌پرداخت باقی بدهی را صفر می‌کند.
	settled := LoanBalanceFor(schedule, []LoanPayment{
		{Seq: 1, Kind: LoanEventInstallment, Amount: 10_000_000},
		{Kind: LoanEventPrepay, Amount: 20_000_000},
	}, "1405/02")
	if !settled.Settled || settled.RemainingDebt != 0 || settled.RemainingInstallments != 0 {
		t.Fatalf("پیش‌پرداخت باید وام را تسویه کند: %+v", settled)
	}

	// پرداخت بیش از بدهی، مانده را منفی نمی‌کند.
	over := LoanBalanceFor(schedule, []LoanPayment{{Kind: LoanEventSettle, Amount: 90_000_000}}, "1405/02")
	if over.RemainingDebt != 0 {
		t.Fatalf("مانده بدهی منفی نشود: %+v", over)
	}
}

// TestLOAN04PayslipCodes نگاشت کد رسمی فیش به نوع وام را می‌سنجد و گزارش صریح انواع
// وامی که کد رسمی ندارند.
func TestLOAN04PayslipCodes(t *testing.T) {
	payment, installment, ok := LoanPayslipCodeFor("emergency")
	if !ok || payment.Code != "500" || installment.Code != "5400" {
		t.Fatalf("وام اضطراری باید کد ۵۰۰ و کسر ۵۴۰۰ بدهد: %q / %q", payment.Code, installment.Code)
	}
	if payment.TitleFa != "وام اضطراری" {
		t.Fatalf("عنوان کد پرداخت باید «وام اضطراری» باشد: %q", payment.TitleFa)
	}
	payment, installment, ok = LoanPayslipCodeFor("socialWork")
	if !ok || payment.Code != "519" || installment.Code != "" {
		t.Fatalf("وام مددکاری باید کد ۵۱۹ و بدون کد کسر باشد: %q / %q", payment.Code, installment.Code)
	}
	payment, installment, ok = LoanPayslipCodeFor("savings")
	if !ok || payment.Code != "502" || installment.Code != "5402" {
		t.Fatalf("وام پس‌انداز باید کد ۵۰۲ و کسر ۵۴۰۲ بدهد: %q / %q", payment.Code, installment.Code)
	}
	if _, _, ok := LoanPayslipCodeFor("housingDeposit"); ok {
		t.Fatal("ودیعهٔ مسکن در جدول رسمی کد ندارد")
	}

	gaps := LoanCodeGaps([]string{"emergency", "housingDeposit", "bank", "savings", "socialWork"})
	if len(gaps) != 2 || gaps[0] != "housingDeposit" || gaps[1] != "bank" {
		t.Fatalf("انواع بی‌کد باید صریح گزارش شوند: %v", gaps)
	}
	if note := LoanCodeNoteFa("housingDeposit"); !strings.Contains(note, "کد رسمی") {
		t.Fatalf("توضیح کد نبودن باید صریح باشد: %q", note)
	}
	if note := LoanCodeNoteFa("socialWork"); !strings.Contains(note, "کد کسر قسط") {
		t.Fatalf("نبود کد کسر قسط باید گفته شود: %q", note)
	}
}

// TestLOAN06History تاریخچهٔ وام را می‌سنجد: پرداخت یک‌باره، اقساط کسرشده/در انتظار،
// تسویه و ماندهٔ جاری پس از هر رخداد.
func TestLOAN06History(t *testing.T) {
	statement, problems := BuildLoanStatement(LoanStatementInput{
		LoanID: "LOAN-0001", LoanType: "emergency", LoanTypeFa: "وام اضطراری",
		TitleFa: "وام اضطراری — نمونه", EmployeeID: "EMP-004", StatusFa: "در حال بازپرداخت",
		Terms: LoanTerms{
			LoanID: "LOAN-0001", LoanType: "emergency", Amount: 30_000_000, Installments: 3,
			ReceivedDateFa: "1404/12/20",
		},
		Payments:          []LoanPayment{{Seq: 1, Kind: LoanEventInstallment, Amount: 10_000_000, PaidDateFa: "1405/01/20"}},
		ReferencePeriodFa: "1405/02",
	})
	if len(problems) != 0 {
		t.Fatalf("صورت حساب معتبر نباید ایراد داشته باشد: %v", problems)
	}
	if !statement.Disbursed {
		t.Fatal("وام با تاریخ پرداخت، پرداخت‌شده شمرده می‌شود")
	}
	if statement.PaymentCode != "500" || statement.InstallmentCode != "5400" {
		t.Fatalf("کدهای صورت حساب درست نیست: %q / %q", statement.PaymentCode, statement.InstallmentCode)
	}
	if len(statement.History) != 4 {
		t.Fatalf("تاریخچه باید ۴ رخداد باشد (پرداخت + ۳ قسط)، %d شد: %+v", len(statement.History), statement.History)
	}
	first := statement.History[0]
	if first.Kind != LoanEventDisburse || first.Amount != 30_000_000 || first.BalanceAfter != 30_000_000 {
		t.Fatalf("رخداد اول باید پرداخت یک‌بارهٔ ۳۰ میلیون با بدهی ۳۰ میلیون باشد: %+v", first)
	}
	if first.CodeFa != "500" || first.DirectionFa != "بستانکار" {
		t.Fatalf("رخداد پرداخت باید کد ۵۰۰ و سوی بستانکار داشته باشد: %+v", first)
	}
	second := statement.History[1]
	if second.Seq != 1 || second.StatusFa != "کسرشده" || second.BalanceAfter != 20_000_000 {
		t.Fatalf("قسط اول باید کسرشده با ماندهٔ ۲۰ میلیون باشد: %+v", second)
	}
	if statement.History[2].StatusFa != "در انتظار کسر" || statement.History[3].StatusFa != "در انتظار کسر" {
		t.Fatalf("اقساط کسرنشده باید «در انتظار کسر» باشند: %+v", statement.History[2:])
	}
	if statement.Balance.RemainingDebt != 20_000_000 || statement.Balance.RemainingInstallments != 2 {
		t.Fatalf("ماندهٔ صورت حساب باید ۲۰ میلیون و ۲ قسط باشد: %+v", statement.Balance)
	}
	if statement.LastPeriodFa != "1405/03" {
		t.Fatalf("آخرین ماه جدول باید 1405/03 باشد، %q شد", statement.LastPeriodFa)
	}

	// تسویهٔ ثبت‌شده روی وام: بدهی صفر و رخداد تسویه در تاریخچه.
	settled, _ := BuildLoanStatement(LoanStatementInput{
		LoanID: "LOAN-0001", LoanType: "emergency",
		Terms: LoanTerms{
			Amount: 30_000_000, Installments: 3, ReceivedDateFa: "1404/12/20",
		},
		Payments:          []LoanPayment{{Seq: 1, Kind: LoanEventInstallment, Amount: 10_000_000}},
		SettledAmount:     20_000_000,
		SettleDateFa:      "1405/02/10",
		ReferencePeriodFa: "1405/03",
	})
	if !settled.Balance.Settled || settled.Balance.RemainingDebt != 0 {
		t.Fatalf("تسویه باید بدهی را صفر کند: %+v", settled.Balance)
	}
	var sawSettle bool
	for _, row := range settled.History {
		if row.Kind == LoanEventSettle {
			sawSettle = true
			if row.Amount != 20_000_000 || row.BalanceAfter != 0 {
				t.Fatalf("رخداد تسویه باید ۲۰ میلیون با ماندهٔ صفر باشد: %+v", row)
			}
		}
	}
	if !sawSettle {
		t.Fatal("رخداد تسویه در تاریخچه نیامد")
	}

	// وام پرداخت‌نشده: هیچ رخداد پرداختی ندارد و ماندهٔ بدهی صفر گزارش می‌شود.
	pending, _ := BuildLoanStatement(LoanStatementInput{
		LoanID: "LOAN-0002", LoanType: "emergency",
		Terms: LoanTerms{Amount: 12_000_000, Installments: 3},
	})
	if pending.Disbursed {
		t.Fatal("وام بدون تاریخ پرداخت نباید پرداخت‌شده شمرده شود")
	}
	if len(pending.History) != 3 || pending.History[0].Kind != LoanEventInstallment {
		t.Fatalf("تاریخچهٔ وام پرداخت‌نشده باید فقط اقساط برنامه‌ای باشد: %+v", pending.History)
	}
	if pending.Balance.RemainingDebt != 0 {
		t.Fatalf("ماندهٔ بدهی وام پرداخت‌نشده باید صفر باشد: %+v", pending.Balance)
	}
}