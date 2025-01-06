package shared

type Email struct {
	ID       string `json:"_id"`
	Body     string `json:"body"`
	Datetime string `json:"datetime"`
	From     string `json:"from"`
	To       string `json:"to"`
	Subject  string `json:"subject"`
}
