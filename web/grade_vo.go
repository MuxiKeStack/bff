package web

type GradeVo struct {
	Regular float64 `json:"regular"` // 平时成绩
	Final   float64 `json:"final"`   // 期末成绩
	Total   float64 `json:"total"`   // 总成绩
	Year    string  `json:"year"`    // 学年
	Term    string  `json:"term"`    // 学期
}

type SignReq struct {
	WantsToSign bool `json:"wants_to_sign"`
}
