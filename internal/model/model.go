package model

type Node struct {
	Host     string `json:"host" yaml:"host"`
	Port     int    `json:"port" yaml:"port"`
	Type     string `json:"type" yaml:"type"`
	UploadAt string `json:"upload_at" yaml:"upload_at"`
	Value1   string `json:"value1" yaml:"value1"`
	Value2   string `json:"value2" yaml:"value2"`
	Value3   string `json:"value3" yaml:"value3"`
	Value4   string `json:"value4" yaml:"value4"`
	Value5   string `json:"value5" yaml:"value5"`
	Value6   string `json:"value6" yaml:"value6"`
	Time     string `json:"time" yaml:"time"`
}

type Platform struct {
	Alias []string `json:"alias" yaml:"alias"`
	Rules []string `json:"rules" yaml:"rules"`
}

type UnlockResponse struct {
	Code int                `json:"code"`
	Data UnlockResponseData `json:"data"`
	Msg  string             `json:"msg"`
}

type UnlockResponseData struct {
	Node     map[string]Node     `json:"node"`
	Platform map[string]Platform `json:"platform"`
}

type UploadPayload struct {
	ID       int      `json:"id"`
	Platform []string `json:"platform"`
}

type UploadResponse struct {
	Code int                `json:"code"`
	Data UploadResponseData `json:"data"`
	Msg  string             `json:"msg"`
}

type UploadResponseData struct {
	NotPlatforms []string `json:"not_platforms"`
}
