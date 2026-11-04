package models

type Pagination struct {
	Total      int64 `json:"total"`
	TotalPages int   `json:"total_pages"`
	Page       int   `json:"page"`
	PageSize   int   `json:"page_size"`
}
