package web

type GradeVo struct {
	Regular float64 `json:"regular"`
	Final   float64 `json:"final"`
	Total   float64 `json:"total"`
	Year    string  `json:"year"`
	Term    string  `json:"term"`
}
