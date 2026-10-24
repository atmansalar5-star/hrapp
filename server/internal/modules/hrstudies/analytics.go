// پیاده‌سازی ماژول «hrstudies» — «analytics».

package hrstudies

import (
	"context"
	"math"
	"net/http"
	"sort"
	"strconv"
	"strings"

	"hrms/internal/contracts"
	"hrms/internal/modules/hrstudies/domain"
	"hrms/internal/platform/auth"
	"hrms/internal/platform/calendar"
	"hrms/internal/platform/httpx"
	"hrms/internal/platform/modulekit"
	"hrms/internal/platform/recordkit"
)

// این فایل دو تبِ اعلام‌شدهٔ ماژول مطالعات را عملیاتی می‌کند:
//
//	GET /api/v1/hrstudies/indices   شاخص‌های سرمایهٔ انسانی + سنجش با هدف
//	GET /api/v1/hrstudies/surveys   نظرسنجی‌ها + تحلیل مشارکت و رضایت
//	GET /api/v1/hrstudies/options   گزینه‌های فیلدهای انتخابی (جهت مطلوب شاخص)
//
// قاعدهٔ معماری: اینجا هیچ قاعدهٔ کسب‌وکاری ساخته نمی‌شود؛ رکوردها از موتور خودِ ماژول
// خوانده می‌شوند و محاسبه (آستانهٔ هدف، دستهٔ مشارکت) از بستهٔ domain می‌آید. پس UI و
// گزارش‌ها همه یک عدد و یک برچسب واحد می‌بینند.
func registerAnalyticsEndpoints(mux *http.ServeMux, deps modulekit.Deps, svc *recordkit.Service) {
	read := func(fn http.HandlerFunc) http.HandlerFunc { return deps.Require("hrstudies", "read", fn) }

	mux.HandleFunc("GET /api/v1/hrstudies/indices", read(func(w http.ResponseWriter, r *http.Request) {
		httpx.WriteJSON(w, http.StatusOK, indicesView(r.Context(), deps, svc))
	}))
	mux.HandleFunc("GET /api/v1/hrstudies/surveys", read(func(w http.ResponseWriter, r *http.Request) {
		httpx.WriteJSON(w, http.StatusOK, surveysView(r.Context(), deps, svc))
	}))
	mux.HandleFunc("GET /api/v1/hrstudies/options", read(func(w http.ResponseWriter, r *http.Request) {
		httpx.WriteJSON(w, http.StatusOK, map[string]any{
			"directions": []map[string]string{
				{"key": string(domain.DirectionHigher), "titleFa": domain.DirectionHigher.LabelFa()},
				{"key": string(domain.DirectionLower), "titleFa": domain.DirectionLower.LabelFa()},
			},
		})
	}))
}

/* ------------------------------ شاخص‌های سرمایهٔ انسانی ------------------------------ */

// indicesView شاخص‌ها را می‌خواند، با هدف می‌سنجد و فهرست + جمع برمی‌گرداند.
//
// ترتیب نمایش عمدی است: شاخص‌های خارج از هدف اول می‌آیند تا مدیر پیش از هر کاری
// مغایرت‌ها را ببیند؛ درون هر سنجه، عنوان فارسی مرتب می‌شود تا ترتیب پایدار بماند.
func indicesView(ctx context.Context, deps modulekit.Deps, svc *recordkit.Service) contracts.HRIndicesView {
	out := contracts.HRIndicesView{
		Items:  []contracts.HRIndicatorView{},
		NoteFa: "سنجش با هدف قاعدهٔ سرور است: انحراف تا ۵٪ در محدودهٔ هدف، تا ۱۵٪ نزدیک هدف و بیشتر از آن خارج از هدف شمرده می‌شود. هدف و مقدار جاری هر دوره در خود رکورد شاخص نگه داشته می‌شود.",
	}
	items := fetchKind(ctx, deps, svc, "hrIndices")
	records := make([]contracts.HRIndicatorView, 0, len(items))
	summary := contracts.HRIndicesSummary{Total: len(items)}
	for _, rec := range items {
		target := num(rec.Fields["target"])
		current := num(rec.Fields["current"])
		dir := domain.ParseDirection(rec.Fields["direction"])
		achieved, gap, verdict := domain.EvaluateIndicator(current, target, dir)
		records = append(records, contracts.HRIndicatorView{
			ID: rec.ID, TitleFa: rec.Title, Status: rec.Status, StatusFa: rec.StatusFa,
			Formula: rec.Fields["formula"], Direction: string(dir), DirectionFa: dir.LabelFa(),
			Unit: rec.Fields["unit"], Target: target, Current: current,
			AchievedPct: achieved, GapPct: gap,
			Assessment: string(verdict), AssessmentFa: verdict.LabelFa(),
			Period: rec.Fields["period"], Owner: rec.Fields["owner"],
		})
		switch verdict {
		case domain.AssessmentInTarget:
			summary.InTarget++
		case domain.AssessmentNearTarget:
			summary.NearTarget++
		default:
			summary.OutOfTarget++
		}
		if rec.Status == "under_review" {
			summary.NeedsReview++
		}
	}
	sort.SliceStable(records, func(i, j int) bool {
		ri, rj := rank(records[i].Assessment), rank(records[j].Assessment)
		if ri != rj {
			return ri < rj
		}
		return records[i].TitleFa < records[j].TitleFa
	})
	if summary.Total > 0 {
		summary.TargetRatePct = round1(float64(summary.InTarget) / float64(summary.Total) * 100)
	}
	summary.NoteFa = "کنترل دوره‌ای شاخص‌های خارج از هدف کار واحد مطالعات است؛ درصد تحقق هدف از نسبت شاخص‌های در محدودهٔ هدف می‌آید."
	out.Items = records
	out.Summary = summary
	return out
}

// rank سختی وضعیت سنجش را برای مرتب‌سازی برمی‌گرداند (کوچک‌تر = فوری‌تر).
func rank(assessment string) int {
	switch assessment {
	case string(domain.AssessmentOutOfTarget):
		return 0
	case string(domain.AssessmentNearTarget):
		return 1
	default:
		return 2
	}
}

/* -------------------------------- نظرسنجی‌ها -------------------------------- */

// surveysView نظرسنجی‌ها را می‌خواند، نرخ مشارکت را در دامنه حساب می‌کند و تحلیل جمعی
// (میانگین مشارکت و رضایت، شمار اقدام‌های اصلاحی) را برمی‌گرداند.
//
// اگر «جامعهٔ آماری» در رکورد ثبت نشده باشد، سرشماری کارکنان فعال سامانه جانشین
// می‌شود تا نرخ مشارکت نظرسنجی‌های قدیمی هم معنادار بماند (و نه صفر گمراه‌کننده).
func surveysView(ctx context.Context, deps modulekit.Deps, svc *recordkit.Service) contracts.HRSurveysView {
	out := contracts.HRSurveysView{
		Items:  []contracts.HRSurveyView{},
		NoteFa: "نرخ مشارکت از نسبت شرکت‌کنندگان به جامعهٔ آماری حساب می‌شود؛ مشارکت از ۷۵٪ خوب، از ۵۰٪ قابل قبول و پایین‌تر از آن ضعیف شمرده می‌شود. جامعهٔ آماری ثبت‌نشده با سرشماری کارکنان فعال جانشین می‌شود.",
	}
	fallback := headcount(ctx, deps)
	var sumParticipation, sumSatisfaction float64
	countedParticipation, countedSatisfaction := 0, 0

	items := fetchKind(ctx, deps, svc, "surveys")
	views := make([]contracts.HRSurveyView, 0, len(items))
	summary := contracts.HRSurveysSummary{Total: len(items)}
	for _, rec := range items {
		participants := int(num(rec.Fields["participants"]))
		population := int(num(rec.Fields["population"]))
		if population <= 0 {
			population = fallback
		}
		rate := domain.ParticipationRate(participants, population)
		band, bandFa := domain.ParticipationBand(rate)
		satisfaction := num(rec.Fields["satisfactionIndex"])
		views = append(views, contracts.HRSurveyView{
			ID: rec.ID, TitleFa: rec.Title, Status: rec.Status, StatusFa: rec.StatusFa,
			Population: population, Participants: participants,
			ParticipationPct: rate, Band: band, BandFa: bandFa,
			SatisfactionIndex: satisfaction, WeakAxis: rec.Fields["weakAxis"],
			StartDate: rec.Fields["startDate"], EndDate: rec.Fields["endDate"],
		})
		summary.Participants += participants
		switch rec.Status {
		case "open":
			summary.Open++
		case "closed":
			summary.Closed++
		}
		if population > 0 {
			sumParticipation += rate
			countedParticipation++
		}
		if satisfaction > 0 {
			sumSatisfaction += satisfaction
			countedSatisfaction++
		}
		if strings.TrimSpace(rec.Fields["actionsFa"]) != "" {
			summary.ActionsIdentified++
		}
	}
	// اجراشده‌ها اول، بعد دورهٔ در جریان و طراحی‌نشده‌ها؛ ترتیب پایدار بر پایهٔ پایان دوره.
	sort.SliceStable(views, func(i, j int) bool {
		ri, rj := surveyRank(views[i].Status), surveyRank(views[j].Status)
		if ri != rj {
			return ri < rj
		}
		if views[i].EndDate != views[j].EndDate {
			return views[i].EndDate > views[j].EndDate
		}
		return views[i].TitleFa < views[j].TitleFa
	})
	if countedParticipation > 0 {
		summary.AvgParticipationPct = round1(sumParticipation / float64(countedParticipation))
	}
	if countedSatisfaction > 0 {
		summary.AvgSatisfactionPct = round1(sumSatisfaction / float64(countedSatisfaction))
	}
	summary.NoteFa = "تحلیل فقط جمع و میانگین می‌گیرد؛ تصمیم دربارهٔ اقدام اصلاحی در جلسهٔ واحد مطالعات گرفته و در همان رکورد نظرسنجی ثبت می‌شود."
	out.Items = views
	out.Summary = summary
	return out
}

// surveyRank ترتیب نمایش نظرسنجی‌ها: اجراشده، در جریان، پیش‌نویس، بایگانی.
func surveyRank(status string) int {
	switch status {
	case "closed":
		return 0
	case "open":
		return 1
	case "draft":
		return 2
	default:
		return 3
	}
}

/* ---------------------------------- کمکی‌ها ---------------------------------- */

// fetchKind رکوردهای یک نوع را با محدودهٔ دسترسی نقش خواننده برمی‌گرداند.
//
// چرا محدوده اینجا اعمال می‌شود: مسیر با deps.Require در سطح ماژول کنترل می‌شود، ولی
// سیاست نقش ممکن است دامنهٔ داده را هم محدود کند (مثلاً مدیر واحد فقط واحد خودش)؛ همان
// کاری که handler عمومی رکوردها می‌کند.
func fetchKind(ctx context.Context, deps modulekit.Deps, svc *recordkit.Service, kind string) []contracts.Record {
	scope := "all"
	if user, ok := auth.UserFrom(ctx); ok && deps.Policy != nil {
		scope = deps.Policy.Scope(user.Role, "hrstudies")
	}
	page, err := svc.List(ctx, contracts.RecordFilter{Kind: kind, Size: 500, Scope: scope})
	if err != nil {
		return nil
	}
	return page.Items
}

// headcount سرشماری کارکنان فعال (و در نبود وضعیت فعال، کل کارکنان) را می‌شمارد.
func headcount(ctx context.Context, deps modulekit.Deps) int {
	reader, ok := deps.Ports.Employees()
	if !ok {
		return 0
	}
	page, err := reader.List(ctx, contracts.EmployeeFilter{Scope: "all", Size: 500})
	if err != nil {
		return 0
	}
	active := 0
	for _, e := range page.Items {
		if e.Status == "active" {
			active++
		}
	}
	if active > 0 {
		return active
	}
	return len(page.Items)
}

// num مقدار عددی یک فیلد رکورد را می‌خواند (رقم فارسی هم پذیرفته می‌شود).
func num(raw string) float64 {
	s := calendar.NormalizeDigits(strings.TrimSpace(raw))
	if s == "" {
		return 0
	}
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0
	}
	return v
}

// round1 به یک رقم اعشار گرد می‌کند تا درصدهای پاسخ یک شکل باشند.
func round1(v float64) float64 { return math.Round(v*10) / 10 }
