// آزمون جریان سرتاسری «اطلاعات پایهٔ تحصیلی و مدرک ملاک حقوق».
//
// چه چیزی سنجیده می‌شود:
//   - سابقهٔ تحصیلی با مقطع، رشته و گرایش و بررسی «مرتبط یا غیرمرتبط بودن تحصیلات»؛
//   - قاعدهٔ دامنه: تأیید مدرک بدون ثبت ارتباط تحصیلات ممکن نیست؛
//   - فرم مخصوص «مدرک ملاک حقوق»: تأیید شورای تعدیل مدرک با شماره و تاریخ صورت‌جلسه،
//     جاری‌سازی، و ممنوعیت جاری‌سازی مدرک «غیرمرتبط»؛
//   - معادل‌سازی «یک مقطع بالاتر» برای ایثارگران و الزام ثبت مقطع معادل‌سازی‌شده؛
//   - خلاصهٔ پروندهٔ تحصیلی (آخرین مدرک، مدرک ملاک، وضعیت ایثارگری) و فرم چاپی آن؛
//   - درخت سازمانی با متصدی، تحصیلات، مدرک ملاک، پایهٔ شخصی و پایهٔ سازمانی سمت.
package bootstrap

import (
	"net/http"
	"strings"
	"testing"

	"hrms/internal/contracts"
)

// educationSummary شکل پاسخ خلاصهٔ پروندهٔ تحصیلی در آزمون.
type educationSummary struct {
	EmployeeID string `json:"employeeId"`
	Records    []struct {
		ID               string `json:"id"`
		Level            string `json:"level"`
		LevelFa          string `json:"levelFa"`
		Field            string `json:"field"`
		Orientation      string `json:"orientation"`
		Relevance        string `json:"relevance"`
		RelevanceFa      string `json:"relevanceFa"`
		RelevancePercent string `json:"relevancePercent"`
		CouncilStatusFa  string `json:"councilStatusFa"`
		Status           string `json:"status"`
		LatestDegree     bool   `json:"latestDegree"`
		IsSalaryBasis    bool   `json:"isSalaryBasis"`
	} `json:"records"`
	Latest *struct {
		ID        string `json:"id"`
		Level     string `json:"level"`
		LevelFa   string `json:"levelFa"`
		Field     string `json:"field"`
		Relevance string `json:"relevance"`
	} `json:"latest"`
	Bases []struct {
		ID               string `json:"id"`
		LevelFa          string `json:"levelFa"`
		EffectiveLevelFa string `json:"effectiveLevelFa"`
		Relevance        string `json:"relevance"`
		VeteranUpgrade   string `json:"veteranUpgrade"`
		UpgradedLevelFa  string `json:"upgradedLevelFa"`
		Status           string `json:"status"`
		CouncilNo        string `json:"councilNo"`
	} `json:"bases"`
	ActiveBasis *struct {
		ID               string `json:"id"`
		LevelFa          string `json:"levelFa"`
		EffectiveLevelFa string `json:"effectiveLevelFa"`
		VeteranUpgrade   string `json:"veteranUpgrade"`
		UpgradedLevelFa  string `json:"upgradedLevelFa"`
		CouncilNo        string `json:"councilNo"`
	} `json:"activeBasis"`
	Veteran struct {
		IsVeteran bool   `json:"isVeteran"`
		Confirmed bool   `json:"confirmed"`
		TypeFa    string `json:"typeFa"`
		NoteFa    string `json:"noteFa"`
	} `json:"veteran"`
	Warnings    []string       `json:"warnings"`
	Summary     map[string]any `json:"summary"`
	BasisNoteFa string         `json:"basisNoteFa"`
}

// createRecord یک رکورد در ماژول داده‌شده می‌سازد و شناسهٔ آن را برمی‌گرداند.
func createRecord(t *testing.T, h http.Handler, token, prefix string, body map[string]any) string {
	t.Helper()
	res := call(t, h, http.MethodPost, prefix+"/records", token, body)
	if res.Code != http.StatusCreated {
		t.Fatalf("ساخت رکورد باید ۲۰۱ بدهد، %d داد: %s", res.Code, res.Body.String())
	}
	var rec contracts.Record
	decodeInto(t, res, &rec)
	if rec.ID == "" {
		t.Fatal("شناسهٔ رکورد ساخته‌شده خالی است")
	}
	return rec.ID
}

// actRecord یک اقدام گردشکار روی رکورد انجام می‌دهد و پاسخ را برمی‌گرداند.
func actRecord(t *testing.T, h http.Handler, token, prefix, id, action string, note string) *struct {
	Code int
	Body string
} {
	t.Helper()
	payload := map[string]any{"action": action}
	if note != "" {
		payload["noteFa"] = note
	}
	res := call(t, h, http.MethodPost, prefix+"/records/"+id+"/action", token, payload)
	return &struct {
		Code int
		Body string
	}{Code: res.Code, Body: res.Body.String()}
}

// TestEducationAndDegreeBasisFlow جریان کامل تحصیلات و تعیین مدرک ملاک حقوق را می‌سنجد.
func TestEducationAndDegreeBasisFlow(t *testing.T) {
	h := newTestServer(t)
	admin := login(t, h, "admin", "demo").AccessToken

	// ---------- گزینه‌های فرم از دامنهٔ سرور ----------
	options := call(t, h, http.MethodGet, "/api/v1/archive/options", admin, nil)
	if options.Code != http.StatusOK {
		t.Fatalf("گزینه‌های فرم تحصیلی باید ۲۰۰ بدهد، %d داد", options.Code)
	}
	var opt struct {
		DegreeLevels []struct {
			Key   string `json:"key"`
			Order int    `json:"order"`
		} `json:"degreeLevels"`
		Relevances []struct {
			Key string `json:"key"`
		} `json:"relevances"`
		CouncilStatuses []struct {
			Key string `json:"key"`
		} `json:"councilStatuses"`
	}
	decodeInto(t, options, &opt)
	if len(opt.DegreeLevels) < 5 || len(opt.Relevances) != 3 || len(opt.CouncilStatuses) != 4 {
		t.Fatalf("فهرست مقطع‌ها/ارتباط/شورا ناقص است: %+v", opt)
	}
	if opt.DegreeLevels[0].Key != "diploma" {
		t.Errorf("نخستین مقطع باید دیپلم باشد: %+v", opt.DegreeLevels[0])
	}

	// ---------- پروندهٔ نمونه: آخرین مدرک و مدرک ملاک ----------
	seeded := educationSummaryOf(t, h, admin, "EMP-004")
	if len(seeded.Records) < 2 {
		t.Fatalf("برای EMP-004 باید چند سابقهٔ تحصیلی باشد، %d آمد", len(seeded.Records))
	}
	if seeded.Latest == nil || seeded.Latest.Level != "master" {
		t.Fatalf("آخرین مدرک باید کارشناسی ارشد باشد: %+v", seeded.Latest)
	}
	if seeded.ActiveBasis == nil || seeded.ActiveBasis.LevelFa == "" {
		t.Fatal("مدرک ملاک حقوق فعال برای EMP-004 انتظار می‌رفت")
	}
	basisFlag := false
	for _, rec := range seeded.Records {
		if rec.IsSalaryBasis {
			basisFlag = true
		}
	}
	if !basisFlag {
		t.Error("یکی از سابقه‌های تحصیلی باید «مدرک ملاک حقوق» نشان داده شود")
	}
	if seeded.BasisNoteFa == "" {
		t.Error("یادداشت ملاک حقوق باید در پاسخ باشد")
	}

	// ---------- سابقهٔ تحصیلی تازه: رابطهٔ تحصیلات با شغل ----------
	eduID := createRecord(t, h, admin, "/api/v1/archive", map[string]any{
		"kind": "educations", "title": "کارشناسی مهندسی صنایع — رضا کاظمی", "employeeId": "EMP-007",
		"fields": map[string]string{
			"level": "bachelor", "field": "مهندسی صنایع", "orientation": "برنامه‌ریزی تولید",
			"university": "دانشگاه صنعتی شریف", "graduationDate": "1396/06/31",
			"relevance": "partially_related", "relevancePercent": "75", "councilStatus": "approved",
		},
	})
	if res := actRecord(t, h, admin, "/api/v1/archive", eduID, "verify", ""); res.Code != http.StatusOK {
		t.Fatalf("تأیید مدارک تحصیلی باید ۲۰۰ بدهد، %d داد: %s", res.Code, res.Body)
	}

	// مدرکی که «ارتباط با شغل» آن ثبت نشده، تأیید نمی‌شود (قاعدهٔ دامنه).
	noRelevance := createRecord(t, h, admin, "/api/v1/archive", map[string]any{
		"kind": "educations", "title": "کارشناسی بدون سنجش ارتباط", "employeeId": "EMP-007",
		"fields": map[string]string{
			"level": "bachelor", "field": "مدیریت", "graduationDate": "1395/06/31",
		},
	})
	res := actRecord(t, h, admin, "/api/v1/archive", noRelevance, "verify", "")
	if res.Code != http.StatusUnprocessableEntity || !strings.Contains(res.Body, "ارتباط") {
		t.Fatalf("تأیید بدون ثبت ارتباط باید ۴۲۲ با پیام ارتباط بدهد: %d — %s", res.Code, res.Body)
	}

	// ---------- فرم مخصوص «مدرک ملاک حقوق» ----------
	// الف) جاری‌سازی مدرک «غیرمرتبط» ممنوع است.
	notRelated := createRecord(t, h, admin, "/api/v1/archive", map[string]any{
		"kind": "degreeBases", "title": "مدرک غیرمرتبط — آزمون", "employeeId": "EMP-007",
		"fields": map[string]string{
			"education": eduID, "level": "bachelor", "field": "مهندسی صنایع",
			"relevance": "not_related", "relevancePercent": "10",
			"councilNo": "SH-1405-001", "councilDate": "1405/07/10", "effectiveDate": "1405/08/01",
			"veteranUpgrade": "no",
		},
	})
	if res := actRecord(t, h, admin, "/api/v1/archive", notRelated, "approve", ""); res.Code != http.StatusOK {
		t.Fatalf("تأیید شورا باید ۲۰۰ بدهد، %d داد: %s", res.Code, res.Body)
	}
	res = actRecord(t, h, admin, "/api/v1/archive", notRelated, "activate", "")
	if res.Code != http.StatusUnprocessableEntity || !strings.Contains(res.Body, "غیرمرتبط") {
		t.Fatalf("جاری‌سازی مدرک غیرمرتبط باید ۴۲۲ بدهد: %d — %s", res.Code, res.Body)
	}

	// ب) برای ایثارگران، جاری‌سازی بدون ثبت «مقطع معادل‌سازی‌شده» ممکن نیست.
	veteranBasis := createRecord(t, h, admin, "/api/v1/archive", map[string]any{
		"kind": "degreeBases", "title": "مدرک ملاک ایثارگری — آزمون", "employeeId": "EMP-009",
		"fields": map[string]string{
			"education": "", "level": "bachelor", "field": "مهندسی صنایع",
			"relevance": "partially_related", "relevancePercent": "75",
			"councilNo": "SH-1405-002", "councilDate": "1405/07/12", "effectiveDate": "1405/08/01",
			"veteranUpgrade": "yes",
		},
	})
	if res := actRecord(t, h, admin, "/api/v1/archive", veteranBasis, "approve", ""); res.Code != http.StatusOK {
		t.Fatalf("تأیید شورا برای مدرک ایثارگری باید ۲۰۰ بدهد: %d — %s", res.Code, res.Body)
	}
	res = actRecord(t, h, admin, "/api/v1/archive", veteranBasis, "activate", "")
	if res.Code != http.StatusUnprocessableEntity || !strings.Contains(res.Body, "معادل") {
		t.Fatalf("جاری‌سازی بدون مقطع معادل‌سازی‌شده باید ۴۲۲ بدهد: %d — %s", res.Code, res.Body)
	}

	// ج) با ثبت مقطع معادل‌سازی‌شده، تصمیم جاری می‌شود و مقطع واقعی یک پله بالاتر است.
	issues := call(t, h, http.MethodPatch, "/api/v1/archive/records/"+veteranBasis, admin, map[string]any{
		"kind": "degreeBases", "title": "مدرک ملاک ایثارگری — آزمون", "employeeId": "EMP-009",
		"fields": map[string]string{
			"education": "", "level": "bachelor", "field": "مهندسی صنایع",
			"relevance": "partially_related", "relevancePercent": "75",
			"councilNo": "SH-1405-002", "councilDate": "1405/07/12", "effectiveDate": "1405/08/01",
			"veteranUpgrade": "yes", "upgradedLevel": "master",
			"salaryGroup": "کارشناسی ارشد (معادل ایثارگری)",
		},
	})
	if issues.Code != http.StatusOK {
		t.Fatalf("ویرایش تصمیم ملاک باید ۲۰۰ بدهد، %d داد: %s", issues.Code, issues.Body.String())
	}
	if res := actRecord(t, h, admin, "/api/v1/archive", veteranBasis, "activate", ""); res.Code != http.StatusOK {
		t.Fatalf("جاری‌سازی با مقطع معادل‌سازی‌شده باید ۲۰۰ بدهد: %d — %s", res.Code, res.Body)
	}

	// د) تأیید شورا بدون شماره و تاریخ صورت‌جلسه ممکن نیست.
	noCouncil := createRecord(t, h, admin, "/api/v1/archive", map[string]any{
		"kind": "degreeBases", "title": "بدون صورت‌جلسه", "employeeId": "EMP-007",
		"fields": map[string]string{
			"education": eduID, "level": "bachelor", "field": "مهندسی صنایع",
			"relevance": "related", "relevancePercent": "100", "effectiveDate": "1405/08/01",
			"veteranUpgrade": "no",
		},
	})
	res = actRecord(t, h, admin, "/api/v1/archive", noCouncil, "approve", "")
	if res.Code != http.StatusUnprocessableEntity || !strings.Contains(res.Body, "صورت‌جلسه") {
		t.Fatalf("تأیید بدون صورت‌جلسه باید ۴۲۲ بدهد: %d — %s", res.Code, res.Body)
	}

	// ---------- خلاصهٔ پروندهٔ تحصیلی EMP-009: ایثارگر و معادل‌سازی ----------
	view := educationSummaryOf(t, h, admin, "EMP-009")
	if !view.Veteran.IsVeteran {
		t.Errorf("پروندهٔ ایثارگری EMP-009 باید در خلاصه دیده شود: %+v", view.Veteran)
	}
	if view.Veteran.NoteFa == "" {
		t.Error("یادداشت وضعیت ایثارگری باید بیاید")
	}
	if view.ActiveBasis == nil {
		t.Fatal("مدرک ملاک جاری EMP-009 انتظار می‌رفت")
	}
	if view.ActiveBasis.VeteranUpgrade != "yes" || view.ActiveBasis.UpgradedLevelFa == "" {
		t.Errorf("معادل‌سازی یک مقطع بالاتر در respuesta انتظار می‌رفت: %+v", view.ActiveBasis)
	}
	// مقطع مبنای پرداخت باید همان مقطع معادل‌سازی‌شده باشد (یک پله بالاتر از خود مدرک).
	if view.ActiveBasis.LevelFa == view.ActiveBasis.EffectiveLevelFa && view.ActiveBasis.UpgradedLevelFa != "" {
		t.Errorf("مقطع مبنای پرداخت باید معادل‌سازی‌شده باشد: %+v", view.ActiveBasis)
	}

	// ---------- فرم چاپی «تعیین مدرک ملاک حقوق» ----------
	document := call(t, h, http.MethodGet, "/api/v1/documents/degree_basis/"+veteranBasis, admin, nil)
	if document.Code != http.StatusOK {
		t.Fatalf("فرم چاپی مدرک ملاک باید ۲۰۰ بدهد، %d داد: %s", document.Code, document.Body.String())
	}
	html := document.Body.String()
	for _, needle := range []string{"فرم تعیین مدرک ملاک حقوق", "شورای تعدیل مدرک", "سابقهٔ تحصیلی ثبت‌شده در پرونده"} {
		if !strings.Contains(html, needle) {
			t.Errorf("فرم چاپی باید «%s» را داشته باشد", needle)
		}
	}

	// ---------- سابقهٔ پرونده: تحصیلات در تب سوابق کارمند می‌آید ----------
	history := call(t, h, http.MethodGet, "/api/v1/history?employeeId=EMP-007", admin, nil)
	if history.Code != http.StatusOK {
		t.Fatalf("سوابق کارمند باید ۲۰۰ بدهد، %d داد", history.Code)
	}
	if !strings.Contains(history.Body.String(), "بایگانی") {
		t.Error("سابقهٔ تحصیلی باید در سوابق پروندهٔ کارمند بیاید")
	}
}

// educationSummaryOf خلاصهٔ پروندهٔ تحصیلی یک کارمند را می‌خواند.
func educationSummaryOf(t *testing.T, h http.Handler, token, employeeID string) educationSummary {
	t.Helper()
	res := call(t, h, http.MethodGet, "/api/v1/archive/education/"+employeeID, token, nil)
	if res.Code != http.StatusOK {
		t.Fatalf("خلاصهٔ تحصیلی %s باید ۲۰۰ بدهد، %d داد: %s", employeeID, res.Code, res.Body.String())
	}
	var view educationSummary
	decodeInto(t, res, &view)
	return view
}

// TestOrgTreeCarriesEducationAndBases درخت سازمانی باید متصدی هر پست را با عکس، تحصیلات،
// مدرک ملاک، پایهٔ شخصی و پایهٔ سازمانی سمت نشان بدهد.
func TestOrgTreeCarriesEducationAndBases(t *testing.T) {
	h := newTestServer(t)
	admin := login(t, h, "admin", "demo").AccessToken

	res := call(t, h, http.MethodGet, "/api/v1/orgstructure/roster", admin, nil)
	if res.Code != http.StatusOK {
		t.Fatalf("ساختار سازمانی باید ۲۰۰ بدهد، %d داد", res.Code)
	}
	var view struct {
		Positions []struct {
			ID      string `json:"id"`
			Code    string `json:"code"`
			OrgBase *struct {
				Grade        string `json:"grade"`
				JobFamily    string `json:"jobFamily"`
				Requirements string `json:"requirements"`
			} `json:"orgBase"`
			Holder *struct {
				EmployeeID string `json:"employeeId"`
				FullName   string `json:"fullName"`
				Veteran    bool   `json:"veteran"`
				Education  *struct {
					LevelFa     string `json:"levelFa"`
					Field       string `json:"field"`
					Relevance   string `json:"relevance"`
					RelevanceFa string `json:"relevanceFa"`
				} `json:"education"`
				DegreeBasis *struct {
					LevelFa          string `json:"levelFa"`
					EffectiveLevelFa string `json:"effectiveLevelFa"`
					CouncilNo        string `json:"councilNo"`
					VeteranUpgrade   bool   `json:"veteranUpgrade"`
				} `json:"degreeBasis"`
				PersonalBase *struct {
					DecreeID   string `json:"decreeId"`
					JobGroup   string `json:"jobGroup"`
					BaseSalary int64  `json:"baseSalary"`
				} `json:"personalBase"`
			} `json:"holder"`
		} `json:"positions"`
		Tree struct {
			Root   string `json:"root"`
			Totals struct {
				Positions int `json:"positions"`
				Filled    int `json:"filled"`
				Units     int `json:"units"`
			} `json:"totals"`
			Units []struct {
				Name      string `json:"name"`
				Positions []struct {
					Code   string `json:"code"`
					Holder *struct {
						EmployeeID string `json:"employeeId"`
					} `json:"holder"`
				} `json:"positions"`
			} `json:"units"`
		} `json:"tree"`
	}
	decodeInto(t, res, &view)

	// درخت باید ریشه و واحد داشته باشد و همهٔ پست‌ها در آن جا افتاده باشند.
	if view.Tree.Root == "" || view.Tree.Totals.Units == 0 {
		t.Fatalf("درخت سازمانی ناقص است: %+v", view.Tree)
	}
	if view.Tree.Totals.Positions != len(view.Positions) {
		t.Errorf("همهٔ پست‌ها باید در درخت بیایند: %d از %d", view.Tree.Totals.Positions, len(view.Positions))
	}
	inTree := map[string]bool{}
	for _, unit := range view.Tree.Units {
		for _, position := range unit.Positions {
			inTree[position.Code] = true
		}
	}
	for _, position := range view.Positions {
		if !inTree[position.Code] {
			t.Errorf("پست %s در درخت سازمانی نیامده است", position.Code)
		}
		if position.OrgBase == nil || position.OrgBase.Grade == "" {
			t.Errorf("پایهٔ سازمانی سمت برای %s باید از شناسنامهٔ پست بیاید", position.Code)
		}
	}

	// متصدی نمونه (EMP-004 روی پست کارشناس ارشد مالی): تحصیلات و مدرک ملاک و پایهٔ شخصی.
	var finance bool
	for _, position := range view.Positions {
		if position.Code != "POS-201" || position.Holder == nil {
			continue
		}
		finance = true
		holder := position.Holder
		if holder.Education == nil || holder.Education.LevelFa == "" {
			t.Errorf("تحصیلات متصدی باید در ساختار سازمانی بیاید: %+v", holder)
		}
		if holder.Education != nil && holder.Education.RelevanceFa == "" {
			t.Errorf("ارتباط تحصیلات با شغل باید نمایش داده شود: %+v", holder.Education)
		}
		if holder.DegreeBasis == nil || holder.DegreeBasis.CouncilNo == "" {
			t.Errorf("مدرک ملاک حقوق متصدی با صورت‌جلسهٔ شورا باید بیاید: %+v", holder.DegreeBasis)
		}
	}
	if !finance {
		t.Fatal("پست کارشناس ارشد مالی با متصدی پیدا نشد")
	}

	// متصدی ایثارگر: پرچم ایثارگری و معادل‌سازی مقطع.
	var veteranSeen bool
	for _, position := range view.Positions {
		if position.Holder == nil {
			continue
		}
		if position.Holder.EmployeeID == "EMP-009" && position.Holder.Veteran {
			veteranSeen = true
			if position.Holder.DegreeBasis != nil && position.Holder.DegreeBasis.VeteranUpgrade &&
				position.Holder.DegreeBasis.EffectiveLevelFa == position.Holder.DegreeBasis.LevelFa {
				t.Errorf("برای ایثارگر، مقطع مبنای پرداخت باید معادل‌سازی‌شده باشد: %+v", position.Holder.DegreeBasis)
			}
		}
	}
	if !veteranSeen {
		t.Error("وضعیت ایثارگری متصدی باید در ساختار سازمانی نمایش داده شود")
	}
}
