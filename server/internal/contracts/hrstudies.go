package contracts

// شاخص‌های سرمایهٔ انسانی و تحلیل نظرسنجی — شکل انتقالی API ماژول مطالعات.
//
// چرا این ساختارها جدا از رکورد عمومی هستند: «مقایسه با هدف» و «نرخ مشارکت» محاسبهٔ
// دامنه‌ای‌اند، نه فیلد خام. سرور این محاسبه را انجام می‌دهد و رابط فقط نمایش می‌دهد؛
// پس هیچ آستانه یا فرمولی در UI سخت نمی‌شود.

// HRIndicatorView یک شاخص سرمایهٔ انسانی همراه با نتیجهٔ سنجش با هدف.
type HRIndicatorView struct {
	ID       string `json:"id"`
	TitleFa  string `json:"titleFa"`
	Status   string `json:"status"`
	StatusFa string `json:"statusFa"`
	// Formula فرمول محاسبهٔ شاخص (متن آزاد، از رکورد می‌آید).
	Formula string `json:"formula,omitempty"`
	// Direction جهت مطلوب: higher (بیشتر بهتر) یا lower (کمتر بهتر).
	Direction   string `json:"direction"`
	DirectionFa string `json:"directionFa"`
	// Unit واحد سنجش (درصد، ریال، ساعت، نفر…).
	Unit string `json:"unit,omitempty"`
	// Target مقدار هدف و Current مقدار جاری دوره.
	Target  float64 `json:"target"`
	Current float64 `json:"current"`
	// AchievedPct نسبت مقدار جاری به هدف (درصد) — برای نمایش پیشرفت.
	AchievedPct float64 `json:"achievedPct"`
	// GapPct انحراف از هدف در جهت نامطلوب (درصد): مثبت یعنی بدتر از هدف.
	GapPct float64 `json:"gapPct"`
	// Assessment وضعیت سنجش با هدف: in_target | near_target | out_of_target.
	Assessment   string `json:"assessment"`
	AssessmentFa string `json:"assessmentFa"`
	// Period دورهٔ سنجش و Owner مالک شاخص.
	Period string `json:"period,omitempty"`
	Owner  string `json:"owner,omitempty"`
}

// HRIndicesSummary جمع شاخص‌ها بر پایهٔ وضعیت سنجش و بازنگری تعریف.
type HRIndicesSummary struct {
	Total       int `json:"total"`
	InTarget    int `json:"inTarget"`
	NearTarget  int `json:"nearTarget"`
	OutOfTarget int `json:"outOfTarget"`
	// NeedsReview شاخص‌هایی که وضعیت رکوردشان «نیازمند بازنگری تعریف» است.
	NeedsReview int `json:"needsReview"`
	// TargetRatePct درصد شاخص‌های در محدودهٔ هدف از کل شاخص‌ها.
	TargetRatePct float64 `json:"targetRatePct"`
	NoteFa        string  `json:"noteFa,omitempty"`
}

// HRIndicesView پاسخ مسیر «شاخص‌های سرمایهٔ انسانی»: فهرست + جمع.
type HRIndicesView struct {
	Items   []HRIndicatorView `json:"items"`
	Summary HRIndicesSummary  `json:"summary"`
	NoteFa  string            `json:"noteFa,omitempty"`
}

// HRSurveyView یک نظرسنجی همراه با تحلیل مشارکت و نتیجه.
type HRSurveyView struct {
	ID       string `json:"id"`
	TitleFa  string `json:"titleFa"`
	Status   string `json:"status"`
	StatusFa string `json:"statusFa"`
	// Population جامعهٔ آماری و Participants تعداد شرکت‌کننده.
	Population   int `json:"population"`
	Participants int `json:"participants"`
	// ParticipationPct نرخ مشارکت (درصد) که در دامنه محاسبه می‌شود.
	ParticipationPct float64 `json:"participationPct"`
	// Band دستهٔ کیفی مشارکت: good | acceptable | weak.
	Band   string `json:"band"`
	BandFa string `json:"bandFa"`
	// SatisfactionIndex شاخص رضایت اعلام‌شدهٔ دوره (درصد) و WeakAxis محور ضعیف.
	SatisfactionIndex float64 `json:"satisfactionIndex"`
	WeakAxis          string  `json:"weakAxis,omitempty"`
	StartDate         string  `json:"startDate,omitempty"`
	EndDate           string  `json:"endDate,omitempty"`
}

// HRSurveysSummary جمع نظرسنجی‌ها.
type HRSurveysSummary struct {
	Total int `json:"total"`
	Open  int `json:"open"`
	// Closed نظرسنجی‌های بسته‌شده.
	Closed              int     `json:"closed"`
	Participants        int     `json:"participants"`
	AvgParticipationPct float64 `json:"avgParticipationPct"`
	AvgSatisfactionPct  float64 `json:"avgSatisfactionPct"`
	ActionsIdentified   int     `json:"actionsIdentified"`
	NoteFa              string  `json:"noteFa,omitempty"`
}

// HRSurveysView پاسخ مسیر «نظرسنجی و جو سازمانی»: فهرست + تحلیل جمعی.
type HRSurveysView struct {
	Items   []HRSurveyView   `json:"items"`
	Summary HRSurveysSummary `json:"summary"`
	NoteFa  string           `json:"noteFa,omitempty"`
}
