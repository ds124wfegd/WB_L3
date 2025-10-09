package entity

type Image struct {
	ID      string            `json:"id"`
	Status  string            `json:"status"`
	Formats map[string]string `json:"formats,omitempty"`
}

type Operation struct {
	Type   string `json:"type"`
	Width  int    `json:"width,omitempty"`
	Height int    `json:"height,omitempty"`
	Text   string `json:"text,omitempty"`
}

type ProcessingTask struct {
	ImageID    string      `json:"image_id"`
	Operations []Operation `json:"operations"`
}

type UploadResponse struct {
	ID     string `json:"id"`
	Status string `json:"status"`
}

type ImageResponse struct {
	ID      string            `json:"id"`
	Status  string            `json:"status"`
	Formats map[string]string `json:"formats,omitempty"`
}
