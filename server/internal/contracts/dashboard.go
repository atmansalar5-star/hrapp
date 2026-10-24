package contracts

// Widget کارت آماری داشبورد: عنوان، مقدار عددی، واحد و روند (دستور ۳-۴ سند طراحی).
type Widget struct {
	Key     string  `json:"key"`
	TitleFa string  `json:"titleFa"`
	Value   float64 `json:"value"`
	Unit    string  `json:"unit,omitempty"`
	Trend   float64 `json:"trend,omitempty"`
}
