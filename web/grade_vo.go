package web

type GradeVo struct {
	Regular float64 `json:"regular"` // 平时成绩
	Final   float64 `json:"final"`   // 期末成绩
	Total   float64 `json:"total"`   // 总成绩
	Year    string  `json:"year"`    // 学年
	Term    string  `json:"term"`    // 学期
}

type GradeChartVo struct {
	Grades [7]struct {
		TotalGrades []float64 `json:"total_grades"`
		Percent     float64   `json:"percent"`
	} `json:"grades"`
	Avg float64 `json:"avg"`
}

type SignReq struct {
	WantsToSign bool `json:"wants_to_sign"`
}
