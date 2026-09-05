package turno

// Turno è volutamente scheletrico: serve a validare la pipeline
// DB -> API -> CI -> deploy, non è la progettazione definitiva del
// dominio (assegnazione, conflitti, disponibilità volontari).
type Turno struct {
	ID           string `json:"id,omitempty"`
	VolontarioID string `json:"volontario_id"`
	Data         string `json:"data"`
	OraInizio    string `json:"ora_inizio"`
	OraFine      string `json:"ora_fine"`
	Stato        string `json:"stato,omitempty"`
}
