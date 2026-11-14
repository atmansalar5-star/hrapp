// مدل جی‌بیست (G20) — بیست شاخص در چهار خانوادهٔ جبران خدمات.
//
// چرا این مدل جدا از مدل ۴پی نوشته می‌شود: مدل ۴پی چهار «عامل» را می‌سنجد و خروجی‌اش باند و
// تصمیم پرداخت است؛ G20 همان چهار عامل را به بیست شاخص قابل اندازه‌گیری می‌شکند تا هم
// امتیاز هر کارمند قابل دفاع و قابل ردیابی باشد و هم بتوان «طبقه»ی کارمند یا کل شرکت را
// تعیین کرد. هر دو مدل مستقل و اختیاری‌اند و هیچ‌کدام محاسبهٔ فیش جاری را عوض نمی‌کنند.
//
// چهار خانواده (همان چهار «پ»):
//
//	P1 person      (پرسن)       دانش، مهارت، شایستگی رفتاری، تجربه، یادگیری‌پذیری
//	P2 position    (پوزیشن)     ارزش شغل، دامنهٔ مسئولیت، پیچیدگی، سختی شرایط، کمیابی بازار
//	P3 performance (پرفورمنس)   تحقق اهداف، کیفیت، بهره‌وری، همکاری، نوآوری
//	P4 partnership (پارتنرشیپ)  تعهد و ماندگاری، مشارکت در تصمیم، انتقال دانش، هم‌راستایی، مالکیت
//
// قاعدهٔ معماری: هر عددی که «سیاست شرکت» است در پارامترها می‌نشیند (`compensation.g20.*`)؛
// وزن بیست شاخص هم دادهٔ پارامتری است (رکورد شاخص‌ها)، نه ثابتِ کد.
package rules

import (
	"math"
	"strconv"
	"strings"
)

// G20Params پارامترهای مدل جی‌بیست (کلیدها با پیشوند `compensation.g20.`).
type G20Params struct {
	// وزن چهار خانوادهٔ امتیاز در امتیاز کل (درصد).
	PersonWeight      float64 `json:"personWeight"`
	PositionWeight    float64 `json:"positionWeight"`
	PerformanceWeight float64 `json:"performanceWeight"`
	PartnershipWeight float64 `json:"partnershipWeight"`
	// IndexMax سقف امتیاز هر شاخص (پیش‌فرض ۱۰۰).
	IndexMax float64 `json:"indexMax"`
	// آستانه‌های طبقه‌بندی: A ممتاز، B پیشرو، C تثبیت‌شده، D در حال رشد، E نیازمند برنامه.
	CutA float64 `json:"cutA"`
	CutB float64 `json:"cutB"`
	CutC float64 `json:"cutC"`
	CutD float64 `json:"cutD"`
	// PartnershipMaxFactor سقف ضریب پارتنرشیپ برای پیوند اختیاری با مدل ۴پی.
	//
	// چرا اینجا: خروجی خانوادهٔ پارتنرشیپ می‌تواند به‌جای امتیاز خام، ضریبی بین ۱٫۰۰ و این
	// سقف بدهد؛ پیش‌فرض ۱٫۰۰ یعنی «بی‌اثر»، پس تا شرکت آن را نخواهد هیچ عدد ۴پی تغییر نمی‌کند.
	PartnershipMaxFactor float64 `json:"partnershipMaxFactor"`
	// PayrollApply آیا خروجی این مدل در پرداخت مصرف شود (پیش‌فرض خاموش، مثل مدل ۴پی).
	PayrollApply bool `json:"payrollApply"`
}

// DefaultG20Params پیش‌فرض‌های آغازین؛ در استقرار واقعی از پارامترهای شرکت خوانده می‌شوند.
func DefaultG20Params() G20Params {
	return G20Params{
		PersonWeight:         25,
		PositionWeight:       25,
		PerformanceWeight:    30,
		PartnershipWeight:    20,
		IndexMax:             100,
		CutA:                 85,
		CutB:                 70,
		CutC:                 55,
		CutD:                 40,
		PartnershipMaxFactor: 1.00,
		PayrollApply:         false,
	}
}

// G20ParamsFromText پارامترهای شرکت (متن) را روی پیش‌فرض می‌نشاند؛ کلید نبوده یا خارج از
// بازهٔ معقول، پیش‌فرض را دست‌نخورده می‌گذارد.
func G20ParamsFromText(values map[string]string) G20Params {
	nums := make(map[string]float64, len(values))
	for k, raw := range values {
		if v, err := strconv.ParseFloat(strings.TrimSpace(raw), 64); err == nil {
			nums[k] = v
		}
	}
	return G20ParamsFromValues(nums)
}

// G20ParamsFromValues همان کار با اعداد آماده (برای آزمون و استفادهٔ درون‌کدی).
func G20ParamsFromValues(values map[string]float64) G20Params {
	p := DefaultG20Params()
	set := func(key string, apply func(float64)) {
		if v, found := values[key]; found {
			apply(v)
		}
	}
	set("compensation.g20.family.person", func(v float64) {
		if v >= 0 && v <= 100 {
			p.PersonWeight = v
		}
	})
	set("compensation.g20.family.position", func(v float64) {
		if v >= 0 && v <= 100 {
			p.PositionWeight = v
		}
	})
	set("compensation.g20.family.performance", func(v float64) {
		if v >= 0 && v <= 100 {
			p.PerformanceWeight = v
		}
	})
	set("compensation.g20.family.partnership", func(v float64) {
		if v >= 0 && v <= 100 {
			p.PartnershipWeight = v
		}
	})
	set("compensation.g20.index.max", func(v float64) {
		if v > 0 && v <= 1000 {
			p.IndexMax = v
		}
	})
	set("compensation.g20.cut.a", func(v float64) {
		if v > 0 && v <= 100 {
			p.CutA = v
		}
	})
	set("compensation.g20.cut.b", func(v float64) {
		if v > 0 && v <= 100 {
			p.CutB = v
		}
	})
	set("compensation.g20.cut.c", func(v float64) {
		if v > 0 && v <= 100 {
			p.CutC = v
		}
	})
	set("compensation.g20.cut.d", func(v float64) {
		if v >= 0 && v <= 100 {
			p.CutD = v
		}
	})
	set("compensation.g20.partnership.maxFactor", func(v float64) {
		if v >= 1 && v <= 2 {
			p.PartnershipMaxFactor = v
		}
	})
	set("compensation.g20.payroll.apply", func(v float64) {
		p.PayrollApply = v != 0
	})
	// وزن خانواده‌ها را به مجموع ۱۰۰ نرمال می‌کنیم تا امتیاز کل معنادار بماند.
	if sum := p.PersonWeight + p.PositionWeight + p.PerformanceWeight + p.PartnershipWeight; sum > 0 {
		p.PersonWeight = p.PersonWeight / sum * 100
		p.PositionWeight = p.PositionWeight / sum * 100
		p.PerformanceWeight = p.PerformanceWeight / sum * 100
		p.PartnershipWeight = p.PartnershipWeight / sum * 100
	}
	return p
}

// ---- چهار خانواده ----

// G20Family کلید و عنوان فارسی یک خانوادهٔ مدل.
type G20Family struct {
	Key     string `json:"key"`
	TitleFa string `json:"titleFa"`
	BriefFa string `json:"briefFa"`
}

// G20Families فهرست چهار خانوادهٔ مدل، به ترتیب ۴پی.
func G20Families() []G20Family {
	return []G20Family{
		{Key: "person", TitleFa: "پرسن (شخص)", BriefFa: "دانش، مهارت، شایستگی رفتاری، تجربه و یادگیری‌پذیری"},
		{Key: "position", TitleFa: "پوزیشن (شغل)", BriefFa: "ارزش شغل، دامنهٔ مسئولیت، پیچیدگی، سختی شرایط و کمیابی بازار"},
		{Key: "performance", TitleFa: "پرفورمنس (عملکرد)", BriefFa: "تحقق اهداف، کیفیت، بهره‌وری، همکاری و نوآوری"},
		{Key: "partnership", TitleFa: "پارتنرشیپ (شراکت)", BriefFa: "تعهد و ماندگاری، مشارکت در تصمیم، انتقال دانش، هم‌راستایی و مالکیت"},
	}
}

// G20FamilyTitleFa عنوان فارسی یک خانواده؛ کلید ناشناخته با احتیاط برگردانده می‌شود.
func G20FamilyTitleFa(key string) string {
	for _, f := range G20Families() {
		if f.Key == key {
			return f.TitleFa
		}
	}
	return key
}

// ---- بیست شاخص پیش‌فرض ----

// G20IndexDef تعریف یک شاخص: کد، خانواده، عنوان، روش سنجش و وزن درون خانواده.
type G20IndexDef struct {
	Code      string
	Family    string
	TitleFa   string
	MeasureFa string
	Weight    float64
}

// G20DefaultIndices بیست شاخص پیش‌فرض — «پیش‌فرضِ پیشنهادی» و قابل ویرایش.
//
// چرا پیش‌فرض و نه فهرست ثابت: فهرست رسمی هر سازمان (و وزن هر شاخص) سیاست شرکت است؛ این
// بیست ردیف فقط نقطهٔ شروع است و در پنل «شاخص‌های جی‌بیست» جایگزین یا بازتنظیم می‌شود.
// برای هر خانواده پنج شاخص و مجموع وزن‌ها ۱۰۰ است.
func G20DefaultIndices() []G20IndexDef {
	return []G20IndexDef{
		{"P1-1", "person", "سطح دانش تخصصی", "مدرک و آزمون دانش تخصصی — امتیاز ۰ تا ۱۰۰", 25},
		{"P1-2", "person", "مهارت فنی و شغلی", "ارزیابی مهارت عملی و مجوزهای شغلی", 25},
		{"P1-3", "person", "شایستگی رفتاری و رهبری", "ارزیابی ۳۶۰ درجهٔ رفتار و رهبری", 20},
		{"P1-4", "person", "سابقه و تجربهٔ مرتبط", "سال‌های تجربهٔ مرتبط با شغل", 15},
		{"P1-5", "person", "یادگیری‌پذیری و انعطاف", "سرعت یادگیری مهارت تازه و سازگاری با تغییر", 15},

		{"P2-1", "position", "ارزش و امتیاز شغل", "امتیاز ارزشیابی شغل (شرح شغل و شناسنامهٔ شغل)", 25},
		{"P2-2", "position", "دامنهٔ مسئولیت و اثر", "گسترهٔ اثر تصمیم‌ها: واحد، اداره یا سازمان", 25},
		{"P2-3", "position", "پیچیدگی و تنوع کار", "تعداد و تنوع تصمیم‌های روزانهٔ شغل", 20},
		{"P2-4", "position", "سختی شرایط محیط کار", "شرایط فیزیکی، نوبت‌کاری و ریسک محیطی", 15},
		{"P2-5", "position", "کمیابی نیرو در بازار", "دشواری تأمین نیروی واجد شرایط در بازار کار", 15},

		{"P3-1", "performance", "تحقق اهداف کمی", "درصد تحقق اهداف مصوب دوره", 30},
		{"P3-2", "performance", "کیفیت کار و نرخ خطا", "نرخ بازکاری، شکایت و خطای قابل انتساب", 25},
		{"P3-3", "performance", "بهره‌وری و مدیریت زمان", "خروجی به ازای زمان و منابع مصرف‌شده", 20},
		{"P3-4", "performance", "همکاری در نتیجهٔ کار", "سهم در نتیجهٔ کار تیمی و پروژه‌های مشترک", 15},
		{"P3-5", "performance", "نوآوری و بهبود فرایند", "پیشنهادها و بهبودهای اجراشده", 10},

		{"P4-1", "partnership", "تعهد سازمانی و ماندگاری", "سابقهٔ ماندگاری و نرخ ترک اختیاری", 25},
		{"P4-2", "partnership", "مشارکت در تصمیم و پیشنهاددهی", "مشارکت در کمیته‌ها و نظام پیشنهادها", 20},
		{"P4-3", "partnership", "انتقال دانش و مربیگری", "جانشین‌پروری و آموزش همکاران", 20},
		{"P4-4", "partnership", "هم‌راستایی با ارزش‌ها و فرهنگ", "ارزیابی هم‌راستایی رفتار با ارزش‌های سازمان", 20},
		{"P4-5", "partnership", "مشارکت مالی و مالکیت", "سهام کارکنان، پاداش بلندمدت و سهیم‌شدن در سود", 15},
	}
}

// G20IndexesByFamily شاخص‌های یک خانواده.
func G20IndexesByFamily(family string) []G20IndexDef {
	out := []G20IndexDef{}
	for _, idx := range G20DefaultIndices() {
		if idx.Family == family {
			out = append(out, idx)
		}
	}
	return out
}

// ---- محاسبهٔ امتیاز ----

// G20Contribution سهم هر خانواده در امتیاز کل (برای نمایش و دفاع از نتیجه).
type G20Contribution struct {
	Key      string  `json:"key"`
	TitleFa  string  `json:"titleFa"`
	Score    float64 `json:"score"`
	Weight   float64 `json:"weight"`
	Share    float64 `json:"share"`
	ShareFa  string  `json:"shareFa"`
	DetailFa string  `json:"detailFa"`
}

// G20Input امتیاز خام چهار خانواده (هر کدام ۰ تا IndexMax).
type G20Input struct {
	Person      float64 `json:"person"`
	Position    float64 `json:"position"`
	Performance float64 `json:"performance"`
	Partnership float64 `json:"partnership"`
}

// G20Result نتیجهٔ محاسبه: امتیاز هر خانواده، امتیاز کل و طبقه.
type G20Result struct {
	Person      float64           `json:"person"`
	Position    float64           `json:"position"`
	Performance float64           `json:"performance"`
	Partnership float64           `json:"partnership"`
	Total       float64           `json:"total"`
	TotalFa     string            `json:"totalFa"`
	SegmentKey  string            `json:"segmentKey"`
	SegmentFa   string            `json:"segmentFa"`
	NoteFa      string            `json:"noteFa"`
	Contrib     []G20Contribution `json:"contributions"`
}

// G20Calc امتیاز کل و طبقه را از امتیاز چهار خانواده می‌سازد.
func G20Calc(in G20Input, p G20Params) G20Result {
	max := p.IndexMax
	if max <= 0 {
		max = 100
	}
	clamp := func(v float64) float64 {
		if v < 0 {
			return 0
		}
		if v > max {
			return max
		}
		return v
	}
	person, position := clamp(in.Person), clamp(in.Position)
	performance, partnership := clamp(in.Performance), clamp(in.Partnership)

	total := person/max*p.PersonWeight + position/max*p.PositionWeight +
		performance/max*p.PerformanceWeight + partnership/max*p.PartnershipWeight

	key, segFa := G20SegmentOf(p, total)
	contrib := []G20Contribution{
		g20Contribution("person", person, p.PersonWeight, max),
		g20Contribution("position", position, p.PositionWeight, max),
		g20Contribution("performance", performance, p.PerformanceWeight, max),
		g20Contribution("partnership", partnership, p.PartnershipWeight, max),
	}
	return G20Result{
		Person: person, Position: position, Performance: performance, Partnership: partnership,
		Total: round1(total), TotalFa: TrimFloatFa(round1(total)),
		SegmentKey: key, SegmentFa: segFa,
		NoteFa:  G20NoteFa(key),
		Contrib: contrib,
	}
}

func g20Contribution(key string, score, weight, max float64) G20Contribution {
	share := score / max * weight
	return G20Contribution{
		Key:      key,
		TitleFa:  G20FamilyTitleFa(key),
		Score:    round1(score),
		Weight:   round1(weight),
		Share:    round1(share),
		ShareFa:  TrimFloatFa(round1(share)),
		DetailFa: "امتیاز " + TrimFloatFa(round1(score)) + " از " + TrimFloatFa(max) + " با وزن " + TrimFloatFa(round1(weight)) + "٪",
	}
}

// G20ScoreFromIndices امتیاز یک خانواده را از امتیاز شاخص‌هایش (با وزن هر شاخص) می‌سازد.
//
// چرا میانگین وزنی و نه جمع: وزن شاخص‌ها درون هر خانواده ۱۰۰ است، پس خروجی هم در همان مقیاس
// ۰ تا IndexMax می‌ماند و مقایسهٔ خانواده‌ها معنادار است.
func G20ScoreFromIndices(scores []float64, weights []float64) float64 {
	if len(scores) == 0 {
		return 0
	}
	var sum, weightSum float64
	for i, s := range scores {
		w := 1.0
		if i < len(weights) && weights[i] > 0 {
			w = weights[i]
		}
		sum += s * w
		weightSum += w
	}
	if weightSum == 0 {
		return 0
	}
	return round1(sum / weightSum)
}

// ---- طبقه‌بندی ----

// G20Segment عنوان و توضیح یک طبقه.
type G20Segment struct {
	Key     string  `json:"key"`
	TitleFa string  `json:"titleFa"`
	From    float64 `json:"from"`
	NoteFa  string  `json:"noteFa"`
}

// G20Segments جدول طبقه‌بندی بر پایهٔ پارامترهای شرکت.
func G20Segments(p G20Params) []G20Segment {
	return []G20Segment{
		{Key: "A", TitleFa: "ممتاز", From: p.CutA, NoteFa: "در چهار خانواده بالای آستانهٔ ممتاز؛ نامزد باند بالا و پاداش بلندمدت"},
		{Key: "B", TitleFa: "پیشرو", From: p.CutB, NoteFa: "عملکرد و شایستگی بالاتر از انتظار؛ نامزد افزایش هدفمند"},
		{Key: "C", TitleFa: "تثبیت‌شده", From: p.CutC, NoteFa: "در محدودهٔ انتظار؛ بازنگری عادی سالانه"},
		{Key: "D", TitleFa: "در حال رشد", From: p.CutD, NoteFa: "زیر انتظار در یک یا چند خانواده؛ برنامهٔ بهبود شش‌ماهه"},
		{Key: "E", TitleFa: "نیازمند برنامه", From: 0, NoteFa: "فاصلهٔ زیاد با انتظار؛ برنامهٔ توسعه و بازآموزی پیش از هر افزایش"},
	}
}

// G20SegmentOf طبقهٔ یک امتیاز کل.
func G20SegmentOf(p G20Params, total float64) (string, string) {
	for _, s := range G20Segments(p) {
		if total >= s.From {
			return s.Key, s.TitleFa
		}
	}
	return "E", "نیازمند برنامه"
}

// G20SegmentTitleFa عنوان فارسی طبقه از کلید.
func G20SegmentTitleFa(key string) string {
	switch strings.ToUpper(strings.TrimSpace(key)) {
	case "A":
		return "ممتاز"
	case "B":
		return "پیشرو"
	case "C":
		return "تثبیت‌شده"
	case "D":
		return "در حال رشد"
	case "E":
		return "نیازمند برنامه"
	}
	return key
}

// G20NoteFa توضیح فارسی طبقه (برای نمایش کنار نتیجه).
func G20NoteFa(key string) string {
	for _, s := range G20Segments(DefaultG20Params()) {
		if s.Key == key {
			return s.NoteFa
		}
	}
	return ""
}

// G20PartnershipFactor ضریب اختیاری پارتنرشیپ برای پیوند با مدل ۴پی.
//
// پیش‌فرض سقف ۱٫۰۰ است، پس تا شرکت سقف را بالا نبرد ضریب همیشه ۱٫۰۰ (بی‌اثر) است و هیچ عددی
// از مدل ۴پی تغییر نمی‌کند.
func G20PartnershipFactor(score float64, p G20Params) float64 {
	max := p.IndexMax
	if max <= 0 {
		max = 100
	}
	if score < 0 {
		score = 0
	}
	if score > max {
		score = max
	}
	ceiling := p.PartnershipMaxFactor
	if ceiling < 1 {
		ceiling = 1
	}
	return round4(1 + (ceiling-1)*score/max)
}

// ---- متون رسمی ----

// G20IndependenceFa متن «استقلال مدل جی‌بیست» — همان مرز سه مدل پرداخت.
func G20IndependenceFa(apply bool) string {
	common := "مدل جی‌بیست (G20) یک لایهٔ سنجش و طبقه‌بندی است، نه موتور پرداخت: " +
		"بیست شاخص را در چهار خانوادهٔ پرسن، پوزیشن، پرفورمنس و پارتنرشیپ می‌سنجد و" +
		" کارمند یا شرکت را طبقه‌بندی می‌کند. مدل حقوق و دستمزد جاری و مدل صندوق نفت" +
		" مستقل و جاری می‌مانند."
	if apply {
		return common + " پارامتر «اعمال در پرداخت» این مدل روشن است، پس نتیجهٔ آن می‌تواند" +
			" مبنای پیشنهاد حکم تازه شود؛ احکام و پرداخت‌های پیشین معتبر می‌مانند و بازنویسی نمی‌شوند."
	}
	return common + " اکنون «اعمال در پرداخت» خاموش است؛ نتیجهٔ آن فقط برای تحلیل و" +
		" طبقه‌بندی به کار می‌رود و هیچ عددی از آن به حکم یا فیش راه نمی‌یابد."
}

// G20InstructionFa راهنمای شروع: ترتیب کارِ درست با مدل جی‌بیست.
func G20InstructionFa() string {
	return "ترتیب کار: نخست شاخص‌های بیست‌گانه و وزن هر خانواده را در پنل شاخص‌ها تثبیت کنید؛" +
		" سپس برای هر کارمند امتیاز چهار خانواده را ثبت کنید (یا از ارزیابی شغل، شایستگی و" +
		" عملکرد سامانه پر کنید)؛ در پایان با «طبقه‌بندی جی‌بیست» امتیاز کل و طبقهٔ هر کارمند" +
		" و همین‌طور طبقهٔ کل شرکت یا یک واحد را ببینید."
}

func round1(v float64) float64 { return math.Round(v*10) / 10 }

func round4(v float64) float64 { return math.Round(v*10000) / 10000 }
