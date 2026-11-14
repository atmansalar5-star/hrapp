// مسیرهای مدل جی‌بیست (G20) در ماژول جبران خدمات.
//
// قاعدهٔ لایه: این فایل فقط داده می‌خواند، به موتور قواعد می‌سپارد و نتیجه را برمی‌گرداند؛
// هیچ فرمولی این‌جا نوشته نمی‌شود و هیچ قاعدهٔ کسب‌وکاری در رابط کاربری نیست.
package compensation

import (
	"context"
	"encoding/json"
	"net/http"
	"sort"
	"strings"

	"hrms/internal/contracts"
	"hrms/internal/modules/compensation/domain"
	"hrms/internal/platform/httpx"
	"hrms/internal/platform/modulekit"
	"hrms/internal/platform/recordkit"
	"hrms/internal/rules"
)

// g20Params پارامترهای مدل جی‌بیست از پروفایل پارامتریک شرکت.
func g20Params(deps modulekit.Deps) rules.G20Params {
	values := map[string]string{}
	for _, p := range deps.Org.Params {
		if strings.HasPrefix(p.Key, "compensation.g20.") {
			values[p.Key] = p.Value
		}
	}
	return rules.G20ParamsFromText(values)
}

// g20IndexRow یک شاخص برای نمایش در پنل.
type g20IndexRow struct {
	RecordID  string  `json:"recordId"`
	Code      string  `json:"code"`
	FamilyKey string  `json:"familyKey"`
	FamilyFa  string  `json:"familyFa"`
	TitleFa   string  `json:"titleFa"`
	MeasureFa string  `json:"measureFa"`
	SourceKey string  `json:"sourceKey"`
	SourceFa  string  `json:"sourceFa"`
	Weight    float64 `json:"weight"`
	WeightFa  string  `json:"weightFa"`
	Status    string  `json:"status"`
	StatusFa  string  `json:"statusFa"`
}

// g20IndexRows شاخص‌های ثبت‌شده (فعال) را می‌خواند؛ اگر هیچ رکوردی نبود، بیست شاخص پیش‌فرض
// موتور قواعد نمایش داده می‌شود تا پنل از روز اول خالی نباشد.
func g20IndexRows(ctx context.Context, svc *recordkit.Service) ([]g20IndexRow, error) {
	page, err := svc.List(ctx, contracts.RecordFilter{
		Module: "compensation", Kind: domain.KindG20Index, Scope: "all", Size: 200,
	})
	if err != nil {
		return nil, err
	}
	rows := make([]g20IndexRow, 0, len(page.Items))
	for _, rec := range page.Items {
		if rec.Status == "archived" {
			continue
		}
		family := rec.Fields["family"]
		weight := numberFromAny(rec.Fields["weightPercent"])
		rows = append(rows, g20IndexRow{
			RecordID: rec.ID, Code: rec.Fields["indexCode"], FamilyKey: family,
			FamilyFa: rules.G20FamilyTitleFa(family), TitleFa: rec.Fields["titleFa"],
			MeasureFa: rec.Fields["measureFa"], SourceKey: rec.Fields["sourceFa"],
			SourceFa: g20SourceTitleFa(rec.Fields["sourceFa"]),
			Weight:   weight, WeightFa: rules.TrimFloatFa(weight),
			Status: rec.Status, StatusFa: g20StatusFa(rec.Status),
		})
	}
	if len(rows) == 0 {
		for _, idx := range rules.G20DefaultIndices() {
			rows = append(rows, g20IndexRow{
				Code: idx.Code, FamilyKey: idx.Family, FamilyFa: rules.G20FamilyTitleFa(idx.Family),
				TitleFa: idx.TitleFa, MeasureFa: idx.MeasureFa,
				Weight: idx.Weight, WeightFa: rules.TrimFloatFa(idx.Weight),
				Status: "default", StatusFa: "پیش‌فرض (ثبت‌نشده)",
			})
		}
	}
	sort.SliceStable(rows, func(i, j int) bool { return rows[i].Code < rows[j].Code })
	return rows, nil
}

// g20SourceTitleFa عنوان فارسی منبع دادهٔ شاخص.
func g20SourceTitleFa(key string) string {
	for _, s := range domain.G20Sources() {
		if s.Key == key {
			return s.TitleFa
		}
	}
	return key
}

// g20StatusFa عنوان فارسی وضعیت شاخص.
func g20StatusFa(status string) string {
	switch status {
	case "draft":
		return "پیش‌نویس"
	case "active":
		return "فعال"
	case "archived":
		return "بایگانی‌شده"
	}
	return status
}

// g20OptionsPayload بدنهٔ پاسخ پنل: پارامترها، چهار خانواده، بیست شاخص، جدول طبقه‌بندی و متون.
func g20OptionsPayload(ctx context.Context, deps modulekit.Deps, svc *recordkit.Service) (map[string]any, error) {
	p := g20Params(deps)
	indices, err := g20IndexRows(ctx, svc)
	if err != nil {
		return nil, err
	}
	segments := rules.G20Segments(p)
	segRows := make([]map[string]any, 0, len(segments))
	for _, s := range segments {
		segRows = append(segRows, map[string]any{
			"key": s.Key, "titleFa": s.TitleFa,
			"fromFa": rules.TrimFloatFa(s.From), "noteFa": s.NoteFa,
		})
	}
	families := make([]map[string]any, 0, len(rules.G20Families()))
	for _, f := range rules.G20Families() {
		families = append(families, map[string]any{
			"key": f.Key, "titleFa": f.TitleFa, "briefFa": f.BriefFa,
			"weightFa": rules.TrimFloatFa(g20FamilyWeight(p, f.Key)),
		})
	}
	return map[string]any{
		"params":           p,
		"families":         families,
		"indices":          indices,
		"indexCount":       len(indices),
		"segments":         segRows,
		"indexMaxFa":       rules.TrimFloatFa(p.IndexMax),
		"independenceFa":   rules.G20IndependenceFa(p.PayrollApply),
		"instructionFa":    rules.G20InstructionFa(),
		"appliedToPayroll": p.PayrollApply,
	}, nil
}

// g20FamilyWeight وزن یک خانواده از پارامترها.
func g20FamilyWeight(p rules.G20Params, key string) float64 {
	switch key {
	case "person":
		return p.PersonWeight
	case "position":
		return p.PositionWeight
	case "performance":
		return p.PerformanceWeight
	case "partnership":
		return p.PartnershipWeight
	}
	return 0
}

// g20AssessmentRequest ورودی محاسبهٔ جی‌بیست از فرم.
type g20AssessmentRequest struct {
	EmployeeID  string  `json:"employeeId"`
	Period      string  `json:"period"`
	Person      float64 `json:"person"`
	Position    float64 `json:"position"`
	Performance float64 `json:"performance"`
	Partnership float64 `json:"partnership"`
}

// toInput ورودی فرم را به ورودی موتور قواعد تبدیل می‌کند.
func (in g20AssessmentRequest) toInput() rules.G20Input {
	return rules.G20Input{
		Person:      in.Person,
		Position:    in.Position,
		Performance: in.Performance,
		Partnership: in.Partnership,
	}
}

// g20AssessmentOf امتیاز چهار خانوادهٔ یک رکورد را می‌خواند.
func g20AssessmentOf(rec contracts.Record) rules.G20Input {
	return rules.G20Input{
		Person:      numberFromAny(rec.Fields["personScore"]),
		Position:    numberFromAny(rec.Fields["positionScore"]),
		Performance: numberFromAny(rec.Fields["performanceScore"]),
		Partnership: numberFromAny(rec.Fields["partnershipScore"]),
	}
}

// g20LatestAssessment آخرین امتیاز جی‌بیست یک کارمند (بر پایهٔ دوره).
func g20LatestAssessment(ctx context.Context, svc *recordkit.Service, employeeID string) (contracts.Record, bool) {
	page, err := svc.List(ctx, contracts.RecordFilter{
		Module: "compensation", Kind: domain.KindG20Assessment, EmployeeID: employeeID, Scope: "all", Size: 50,
	})
	if err != nil || len(page.Items) == 0 {
		return contracts.Record{}, false
	}
	records := page.Items
	sort.SliceStable(records, func(i, j int) bool {
		return records[i].Fields["period"] < records[j].Fields["period"]
	})
	return records[len(records)-1], true
}

// g20BoardRow یک سطر طبقه‌بندی: کارمند، امتیاز چهار خانواده، امتیاز کل و طبقه.
type g20BoardRow struct {
	EmployeeID     string  `json:"employeeId"`
	FullName       string  `json:"fullName"`
	DepartmentName string  `json:"departmentName"`
	Period         string  `json:"period"`
	HasAssessment  bool    `json:"hasAssessment"`
	PersonFa       string  `json:"personFa"`
	PositionFa     string  `json:"positionFa"`
	PerformanceFa  string  `json:"performanceFa"`
	PartnershipFa  string  `json:"partnershipFa"`
	Total          float64 `json:"total"`
	TotalFa        string  `json:"totalFa"`
	SegmentKey     string  `json:"segmentKey"`
	SegmentFa      string  `json:"segmentFa"`
	NoteFa         string  `json:"noteFa"`
	StatusFa       string  `json:"statusFa"`
}

// g20BoardRows همهٔ کارکنان را با آخرین امتیاز جی‌بیستشان ترکیب می‌کند.
func g20BoardRows(ctx context.Context, deps modulekit.Deps, svc *recordkit.Service, query string) ([]g20BoardRow, error) {
	reader, ok := deps.Ports.Employees()
	if !ok {
		return nil, nil
	}
	page, err := reader.List(ctx, contracts.EmployeeFilter{Scope: "all", Size: 300, Query: query})
	if err != nil {
		return nil, err
	}
	p := g20Params(deps)
	rows := make([]g20BoardRow, 0, len(page.Items))
	for _, emp := range page.Items {
		row := g20BoardRow{
			EmployeeID: emp.ID, FullName: emp.FullName, DepartmentName: emp.DepartmentName,
			SegmentKey: "-", SegmentFa: "بدون سنجش", NoteFa: "برای این کارمند هنوز امتیاز جی‌بیست ثبت نشده است.",
			StatusFa: "بدون رکورد",
		}
		if rec, found := g20LatestAssessment(ctx, svc, emp.ID); found {
			res := rules.G20Calc(g20AssessmentOf(rec), p)
			row.HasAssessment = true
			row.Period = rec.Fields["period"]
			row.PersonFa = rules.TrimFloatFa(res.Person)
			row.PositionFa = rules.TrimFloatFa(res.Position)
			row.PerformanceFa = rules.TrimFloatFa(res.Performance)
			row.PartnershipFa = rules.TrimFloatFa(res.Partnership)
			row.Total = res.Total
			row.TotalFa = res.TotalFa
			row.SegmentKey = res.SegmentKey
			row.SegmentFa = res.SegmentFa
			row.NoteFa = res.NoteFa
			row.StatusFa = g20StatusFa(rec.Status)
		}
		rows = append(rows, row)
	}
	sort.SliceStable(rows, func(i, j int) bool {
		if rows[i].HasAssessment != rows[j].HasAssessment {
			return rows[i].HasAssessment
		}
		return rows[i].Total > rows[j].Total
	})
	return rows, nil
}

// g20CompanySummary طبقهٔ کل شرکت: میانگین امتیاز کارکنان سنجیده‌شده.
type g20CompanySummary struct {
	Assessed     int              `json:"assessed"`
	Unassessed   int              `json:"unassessed"`
	Average      float64          `json:"average"`
	AverageFa    string           `json:"averageFa"`
	SegmentKey   string           `json:"segmentKey"`
	SegmentFa    string           `json:"segmentFa"`
	NoteFa       string           `json:"noteFa"`
	CountBySeg   map[string]int   `json:"countBySegment"`
	CountBySegFa []map[string]any `json:"segments"`
}

// g20CompanyOf میانگین و طبقهٔ شرکت را می‌سازد.
func g20CompanyOf(rows []g20BoardRow, p rules.G20Params) g20CompanySummary {
	var sum float64
	assessed := 0
	counts := map[string]int{"A": 0, "B": 0, "C": 0, "D": 0, "E": 0}
	for _, r := range rows {
		if !r.HasAssessment {
			continue
		}
		assessed++
		sum += r.Total
		counts[r.SegmentKey]++
	}
	out := g20CompanySummary{
		Assessed: assessed, Unassessed: len(rows) - assessed,
		CountBySeg: counts, AverageFa: rules.TrimFloatFa(0),
		SegmentKey: "-", SegmentFa: "بدون داده",
		NoteFa: "برای طبقه‌بندی شرکت، دست‌کم یک کارمند باید امتیاز جی‌بیست داشته باشد.",
	}
	if assessed == 0 {
		return out
	}
	avg := sum / float64(assessed)
	key, segFa := rules.G20SegmentOf(p, avg)
	out.Average = float64(int(avg*10+0.5)) / 10
	out.AverageFa = rules.TrimFloatFa(out.Average)
	out.SegmentKey = key
	out.SegmentFa = segFa
	out.NoteFa = "میانگین امتیاز " + rules.IntFa(assessed) + " کارمند سنجیده‌شده؛ طبقهٔ شرکت بر همین میانگین تعیین می‌شود."
	order := []string{"A", "B", "C", "D", "E"}
	out.CountBySegFa = make([]map[string]any, 0, len(order))
	for _, k := range order {
		out.CountBySegFa = append(out.CountBySegFa, map[string]any{
			"key": k, "titleFa": rules.G20SegmentTitleFa(k), "countFa": rules.IntFa(counts[k]),
		})
	}
	return out
}

// registerG20Endpoints مسیرهای مدل جی‌بیست را ثبت می‌کند.
func registerG20Endpoints(mux *http.ServeMux, deps modulekit.Deps, svc *recordkit.Service) {
	mux.HandleFunc("/api/v1/compensation/g20-options", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			httpx.WriteErrorMsg(w, http.StatusMethodNotAllowed, contracts.CodeBadRequest, "فقط GET مجاز است.", nil)
			return
		}
		out, err := g20OptionsPayload(r.Context(), deps, svc)
		if err != nil {
			httpx.WriteErrorMsg(w, http.StatusInternalServerError, contracts.CodeInternal, err.Error(), nil)
			return
		}
		httpx.WriteJSON(w, http.StatusOK, out)
	})

	mux.HandleFunc("/api/v1/compensation/g20-defaults", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			httpx.WriteErrorMsg(w, http.StatusMethodNotAllowed, contracts.CodeBadRequest, "فقط GET مجاز است.", nil)
			return
		}
		out, err := g20OptionsPayload(r.Context(), deps, svc)
		if err != nil {
			httpx.WriteErrorMsg(w, http.StatusInternalServerError, contracts.CodeInternal, err.Error(), nil)
			return
		}
		employeeID := strings.TrimSpace(r.URL.Query().Get("employeeId"))
		out["employeeId"] = employeeID
		if employeeID != "" {
			if rec, found := g20LatestAssessment(r.Context(), svc, employeeID); found {
				in := g20AssessmentOf(rec)
				p := g20Params(deps)
				out["assessment"] = map[string]any{
					"recordId": rec.ID, "period": rec.Fields["period"], "status": rec.Status,
					"person": rules.TrimFloatFa(in.Person), "position": rules.TrimFloatFa(in.Position),
					"performance": rules.TrimFloatFa(in.Performance),
					"partnership": rules.TrimFloatFa(in.Partnership),
					"result":      rules.G20Calc(in, p),
				}
			}
		}
		httpx.WriteJSON(w, http.StatusOK, out)
	})

	mux.HandleFunc("/api/v1/compensation/g20-calc", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			httpx.WriteErrorMsg(w, http.StatusMethodNotAllowed, contracts.CodeBadRequest, "فقط POST مجاز است.", nil)
			return
		}
		var in g20AssessmentRequest
		if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
			httpx.WriteErrorMsg(w, http.StatusBadRequest, contracts.CodeBadRequest, "ورودی محاسبهٔ جی‌بیست خوانده نشد.", nil)
			return
		}
		p := g20Params(deps)
		res := rules.G20Calc(in.toInput(), p)
		httpx.WriteJSON(w, http.StatusOK, map[string]any{
			"employeeId": in.EmployeeID, "period": in.Period,
			"result": res, "params": p,
			"independenceFa": rules.G20IndependenceFa(p.PayrollApply),
		})
	})

	mux.HandleFunc("/api/v1/compensation/g20-board", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			httpx.WriteErrorMsg(w, http.StatusMethodNotAllowed, contracts.CodeBadRequest, "فقط GET مجاز است.", nil)
			return
		}
		query := strings.TrimSpace(r.URL.Query().Get("q"))
		rows, err := g20BoardRows(r.Context(), deps, svc, query)
		if err != nil {
			httpx.WriteErrorMsg(w, http.StatusInternalServerError, contracts.CodeInternal, err.Error(), nil)
			return
		}
		p := g20Params(deps)
		company := g20CompanyOf(rows, p)
		httpx.WriteJSON(w, http.StatusOK, map[string]any{
			"rows": rows, "count": len(rows), "company": company,
			"segments":         g20SegmentRows(p),
			"independenceFa":   rules.G20IndependenceFa(p.PayrollApply),
			"instructionFa":    rules.G20InstructionFa(),
			"appliedToPayroll": p.PayrollApply,
		})
	})
}

// g20SegmentRows جدول طبقه‌بندی برای نمایش.
func g20SegmentRows(p rules.G20Params) []map[string]any {
	segs := rules.G20Segments(p)
	out := make([]map[string]any, 0, len(segs))
	for _, s := range segs {
		out = append(out, map[string]any{
			"key": s.Key, "titleFa": s.TitleFa,
			"fromFa": rules.TrimFloatFa(s.From), "noteFa": s.NoteFa,
		})
	}
	return out
}
