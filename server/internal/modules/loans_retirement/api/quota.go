// پایانهٔ HTTP ماژول «loans_retirement»: پذیرش JSON، اعتبارسنجی ورودی و پاسخ استاندارد httpx.

package api

import (
	"context"
	"net/http"
	"sort"
	"strconv"
	"strings"

	"hrms/internal/contracts"
	"hrms/internal/modules/loans_retirement/domain"
	"hrms/internal/platform/calendar"
	"hrms/internal/platform/httpx"
)

// QuotaDashboard داشبورد «تخصیص سهمیهٔ تسهیلات».
//
// سه برش می‌سازد و همه را از دو منبع واقعی می‌خواند:
//   - جدول سهمیه و مراحل ارسال، از کاتالوگ دامنه (سیاست سازمان)؛
//   - رکوردهای وام و پروندهٔ کارکنان، از انبار ماژول و پورت کارکنان.
//
// رابط وب هیچ درصد یا سهمیه‌ای نمی‌سازد: هر عددی که در نمودارها دیده می‌شود یا از این
// پاسخ می‌آید یا مستقیم از رکوردهای همان صفحه.
func (h *Handler) QuotaDashboard(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	page, err := h.svc.List(ctx, contracts.RecordFilter{
		Module: "loans_retirement", Kind: "loans", Scope: "all", Page: 1, Size: 1000,
	})
	if err != nil {
		httpx.WriteErrorMsg(w, http.StatusInternalServerError, contracts.CodeInternal, err.Error(), nil)
		return
	}
	groups := h.employmentGroups(ctx)

	out := contracts.LoanQuotaDashboard{
		QuotaTotal: float64(domain.QuotaTotalAmount),
		Eligible:   domain.QuotaEligibleTotal(),
		NoteFa:     "سهمیه و سرشماری مشمولین از جدول سهمیهٔ مصوب دامنه می‌آید؛ تعداد متقاضیان، مبالغ، مرحلهٔ ارسال و سهم شرکتی/ارکان ثالث از رکوردهای واقعی وام و پروندهٔ کارکنان محاسبه می‌شود.",
	}

	// انباشتگر هر مدیریت، به ترتیب کاتالوگ (مدیریت‌های بدون درخواست هم صفر می‌مانند).
	type unitAgg struct {
		applicants map[string]bool
		pending    int
		sent       int
		rejected   int
		records    int
		companyN   int
		thirdN     int
		requested  float64
		sentAmount float64
		companyAmt float64
		thirdAmt   float64
	}
	units := domain.QuotaUnits()
	aggs := make([]unitAgg, len(units))
	index := map[string]int{}
	for i, u := range units {
		aggs[i] = unitAgg{applicants: map[string]bool{}}
		index[u.Key] = i
	}

	type groupAgg struct {
		count   int
		amount  float64
		sentN   int
		sentAmt float64
	}
	groupAggs := map[string]*groupAgg{}
	for _, g := range domain.EmploymentGroups() {
		groupAggs[g] = &groupAgg{}
	}

	type stageAgg struct {
		count  int
		amount float64
		compN  int
		thirdN int
	}
	stageAggs := map[string]*stageAgg{}
	for _, s := range domain.QuotaStages() {
		stageAggs[s.Key] = &stageAgg{}
	}

	type tierAgg struct {
		count    int
		amount   float64
		compN    int
		thirdN   int
		compAmt  float64
		thirdAmt float64
	}
	tiers := map[float64]*tierAgg{}

	for _, rec := range page.Items {
		f := rec.Fields
		amount := quotaFloat(f["amount"])
		group := groups[rec.EmployeeID]
		if group == "" {
			group = domain.EmploymentGroupCompany
		}
		sent := domain.SentToFinance(rec.Status)
		rejected := stageIs(rec.Status, "rejected")

		if gi, ok := index[strings.TrimSpace(f["management"])]; ok {
			a := &aggs[gi]
			a.records++
			a.requested += amount
			a.applicants[rec.EmployeeID] = true
			switch {
			case rejected:
				a.rejected++
			case sent:
				a.sent++
				a.sentAmount += amount
			default:
				a.pending++
			}
			if group == domain.EmploymentGroupThirdParty {
				a.thirdN++
				a.thirdAmt += amount
			} else {
				a.companyN++
				a.companyAmt += amount
			}
		}

		if ga := groupAggs[group]; ga != nil {
			ga.count++
			ga.amount += amount
			if sent {
				ga.sentN++
				ga.sentAmt += amount
			}
		}

		if sa := stageAggs[domain.QuotaStageOf(rec.Status).Key]; sa != nil {
			sa.count++
			sa.amount += amount
			if group == domain.EmploymentGroupThirdParty {
				sa.thirdN++
			} else {
				sa.compN++
			}
		}

		rate, ok := feeRateOf(f)
		if ok {
			ta := tiers[rate]
			if ta == nil {
				ta = &tierAgg{}
				tiers[rate] = ta
			}
			ta.count++
			ta.amount += amount
			if group == domain.EmploymentGroupThirdParty {
				ta.thirdN++
				ta.thirdAmt += amount
			} else {
				ta.compN++
				ta.compAmt += amount
			}
		}

		out.Requested += amount
		if sent {
			out.Sent++
			out.SentAmount += amount
		} else if rejected {
			out.Rejected++
		} else {
			out.Pending++
		}
	}

	for i, u := range units {
		a := aggs[i]
		quotaAmount := float64(domain.QuotaTotalAmount) * u.SharePercent / 100
		row := contracts.LoanQuotaUnitRow{
			UnitKey:          u.Key,
			UnitFa:           u.TitleFa,
			SharePercent:     u.SharePercent,
			QuotaAmount:      quotaAmount,
			Eligible:         u.Eligible,
			Applicants:       len(a.applicants),
			Pending:          a.pending,
			Sent:             a.sent,
			Rejected:         a.rejected,
			RecordsCount:     a.records,
			CompanyCount:     a.companyN,
			ThirdPartyCount:  a.thirdN,
			RequestedAmount:  a.requested,
			SentAmount:       a.sentAmount,
			CompanyAmount:    a.companyAmt,
			ThirdPartyAmount: a.thirdAmt,
			ClaimPercent:     percent(a.requested, quotaAmount),
			CoveragePercent:  percent(float64(len(a.applicants)), float64(u.Eligible)),
		}
		out.Units = append(out.Units, row)
		out.Applicants += row.Applicants
	}

	out.Stages = make([]contracts.LoanQuotaStageRow, 0, len(domain.QuotaStages()))
	for _, s := range domain.QuotaStages() {
		a := stageAggs[s.Key]
		out.Stages = append(out.Stages, contracts.LoanQuotaStageRow{
			StageKey: s.Key, StageFa: s.TitleFa, SentToFinance: s.SentToFinance,
			Count: a.count, Amount: a.amount, CompanyCount: a.compN, ThirdPartyCount: a.thirdN,
		})
	}

	var companyAmt, thirdAmt, companySent, thirdSent float64
	for _, g := range domain.EmploymentGroups() {
		a := groupAggs[g]
		if a == nil {
			continue
		}
		row := contracts.LoanQuotaGroupRow{
			GroupKey: g, GroupFa: domain.EmploymentGroupTitleFa(g),
			Count: a.count, Amount: a.amount, SentCount: a.sentN, SentAmount: a.sentAmt,
			SharePercent: percent(a.amount, out.Requested),
		}
		if g == domain.EmploymentGroupThirdParty {
			thirdAmt, thirdSent = a.amount, a.sentAmt
		} else {
			companyAmt, companySent = a.amount, a.sentAmt
		}
		out.Groups = append(out.Groups, row)
	}
	sentTotal := companySent + thirdSent
	for i := range out.Groups {
		if out.Groups[i].GroupKey == domain.EmploymentGroupThirdParty {
			out.Groups[i].SentSharePercent = percent(thirdSent, sentTotal)
		} else {
			out.Groups[i].SentSharePercent = percent(companySent, sentTotal)
		}
	}
	out.CompanyPct = percent(companyAmt, out.Requested)
	out.ThirdPct = percent(thirdAmt, out.Requested)

	rates := make([]float64, 0, len(tiers))
	for r := range tiers {
		rates = append(rates, r)
	}
	sort.Float64s(rates)
	out.FeeTiers = make([]contracts.LoanQuotaFeeTierRow, 0, len(rates))
	for _, r := range rates {
		a := tiers[r]
		out.FeeTiers = append(out.FeeTiers, contracts.LoanQuotaFeeTierRow{
			FeeRatePercent: r, Count: a.count, Amount: a.amount,
			CompanyCount: a.compN, ThirdPartyCount: a.thirdN,
			CompanyAmount: a.compAmt, ThirdPartyAmount: a.thirdAmt,
		})
	}

	httpx.WriteJSON(w, http.StatusOK, out)
}

// employmentGroups نگاشت شناسهٔ کارمند به گروه کارگزینی (شرکتی / ارکان ثالث) از پورت
// کارکنان. اگر پورت در دسترس نباشد، همهٔ وام‌ها «شرکتی» شمرده می‌شوند.
func (h *Handler) employmentGroups(ctx context.Context) map[string]string {
	out := map[string]string{}
	if h.deps.Ports == nil {
		return out
	}
	reader, ok := h.deps.Ports.Employees()
	if !ok {
		return out
	}
	page, err := reader.List(ctx, contracts.EmployeeFilter{Scope: "all", Page: 1, Size: 1000})
	if err != nil {
		return out
	}
	for _, e := range page.Items {
		out[e.ID] = domain.EmploymentGroupOf(e.EmploymentType)
	}
	return out
}

// stageIs آیا وضعیت رکورد در مرحلهٔ موردنظر قرار می‌گیرد.
func stageIs(status, stageKey string) bool { return domain.QuotaStageOf(status).Key == stageKey }

// quotaFloat ارقام فارسی/جداکننده‌دار فیلد مبلغ را به عدد تبدیل می‌کند.
func quotaFloat(v string) float64 {
	v = calendar.NormalizeDigits(strings.TrimSpace(v))
	v = strings.NewReplacer(",", "", "٫", ".").Replace(v)
	f, err := strconv.ParseFloat(v, 64)
	if err != nil {
		return 0
	}
	return f
}

// feeRateOf نرخ کارمزد مؤثر رکورد: مقدار صریح فیلد، وگرنه نرخ پیش‌فرض نوع وام از
// کاتالوگ دامنه. ok=false یعنی نرخ قابل تعیین نیست و رکورد در هیچ طبقه‌ای نمی‌افتد.
func feeRateOf(f map[string]string) (float64, bool) {
	if v := strings.TrimSpace(f["feeRate"]); v != "" {
		if parsed, err := strconv.ParseFloat(v, 64); err == nil {
			return parsed, true
		}
	}
	if t, ok := domain.LoanTypeByKey(strings.TrimSpace(f["loanType"])); ok {
		return t.FeeRatePercent, true
	}
	return 0, false
}

// percent نسبت را به درصد با یک رقم اعشار تبدیل می‌کند؛ مخرج صفر یعنی صفر.
func percent(part, total float64) float64 {
	if total <= 0 {
		return 0
	}
	return float64(int(part/total*1000+0.5)) / 10
}
