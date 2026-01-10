package ds

// PaginationInfo представляет метаданные пагинации
type PaginationInfo struct {
	Page       int   `json:"page"`
	PageSize   int   `json:"page_size"`
	Total      int64 `json:"total"`
	TotalPages int   `json:"total_pages"`
}

// IntervalFiltersInfo представляет примененные фильтры
type IntervalFiltersInfo struct {
	Title   string  `json:"title,omitempty"`
	ToneMin float64 `json:"tone_min,omitempty"`
	ToneMax float64 `json:"tone_max,omitempty"`
}

// QueryStats представляет статистику выполнения запроса
type QueryStats struct {
	ExecutionTimeMs int64  `json:"execution_time_ms"`
	IndexUsed       bool   `json:"index_used"`
	QueryPlan       string `json:"query_plan,omitempty"`
}

// PaginatedIntervalsResponse представляет ответ с пагинированными интервалами
type PaginatedIntervalsResponse struct {
	Data       []Interval           `json:"data"`
	Pagination PaginationInfo       `json:"pagination"`
	Filters    *IntervalFiltersInfo `json:"filters,omitempty"`
	Stats      *QueryStats          `json:"stats,omitempty"`
}
