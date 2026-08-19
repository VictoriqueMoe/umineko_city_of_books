package dto

type (
	LinkPreviewResponse struct {
		URL         string `json:"url"`
		Type        string `json:"type"`
		Title       string `json:"title,omitempty"`
		Description string `json:"description,omitempty"`
		Image       string `json:"image,omitempty"`
		SiteName    string `json:"site_name,omitempty"`
		VideoID     string `json:"video_id,omitempty"`
	}
)
