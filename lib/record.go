package lib

// Bu dosya panel veri modeli yardımcılarını içerir.

// Field, HTML tasarımlarındaki bir yer tutucu alanını temsil eder.
// Ad, değer ve "SSE ile canlı güncelle" seçeneği vardır.
type Field struct {
	Name       string
	Value      string
	SSEEnabled bool
}

// Record, panelin yönettiği bir tasarım kaydını (screen_overlay veya widget) temsil eder.
type Record struct {
	ID     string
	Name   string
	Type   string // "screen_overlay" veya "widget"
	Fields []Field
}

// TypeLabel türün panelde görünen Türkçe adını döndürür.
func TypeLabel(typ string) string {
	switch typ {
	case "screen_overlay":
		return "Screen Overlay"
	case "widget":
		return "Widget"
	default:
		return typ
	}
}
