package dto

// BrandConfigRequest is the request body for saving system brand settings.
type BrandConfigRequest struct {
	SystemName string `json:"system_name" binding:"required,max=128"`
	LogoURL    string `json:"logo_url" binding:"required,max=512"`
}

// BrandConfigResponse is the public system brand configuration.
type BrandConfigResponse struct {
	ID         string `json:"id"`
	SystemName string `json:"system_name"`
	LogoURL    string `json:"logo_url"`
	CreatedAt  string `json:"created_at"`
	UpdatedAt  string `json:"updated_at"`
}

// BrandLogoUploadResponse is returned after uploading a brand logo.
type BrandLogoUploadResponse struct {
	LogoURL string `json:"logo_url"`
}
