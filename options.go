package dyno

type Option struct {
	ID       string `json:"id"`
	Conf     string `json:"conf"`
	LogLevel string `json:"log_level"`
	KWArgs   string `json:"kwargs"`
}
