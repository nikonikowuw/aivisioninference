// Package dto defines Data Transfer Objects for request/response
// validation and serialization.
package dto

// Response is the standard API response wrapper.
type Response struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// PageData is the paginated response data.
type PageData struct {
	List     interface{} `json:"list"`
	Total    int64       `json:"total"`
	Page     int         `json:"page"`
	PageSize int         `json:"page_size"`
}

// PageRequest is the common pagination request.
type PageRequest struct {
	Page     int    `form:"page" json:"page"`
	PageSize int    `form:"page_size" json:"page_size"`
	Sort     string `form:"sort" json:"sort"`
	Order    string `form:"order" json:"order"`
}

// GetPage returns the current page, defaulting to 1 if less than 1.
func (p *PageRequest) GetPage() int {
	if p.Page < 1 {
		p.Page = 1
	}
	return p.Page
}

// GetPageSize returns the page size, clamped between 1 and 100.
func (p *PageRequest) GetPageSize() int {
	if p.PageSize < 1 || p.PageSize > 100 {
		p.PageSize = 20
	}
	return p.PageSize
}
