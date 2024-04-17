package web

type EvaluationSaveReq struct {
	Id          int64    `json:"id"`
	CourseId    int64    `json:"course_id"`
	StarRating  uint8    `json:"star_rating"` // 1，2，3，4，5
	Content     string   `json:"content"`     // 评价的内容
	Assessments []string `json:"assessments"` // 考核方式，支持多选
	Features    []string `json:"features"`    // 课程特点，支持多选
	Status      string   `json:"status"`      // 可见性：Public/Private
}

type EvaluationUpdateStatusReq struct {
	Status string `json:"status"`
}

type EvaluationListRecentReq struct {
	CurEvaluationId int64  `form:"cur_evaluation_id"`
	Limit           int64  `form:"limit"`
	Property        string `form:"property"`
}

type EvaluationVo struct {
	Id          int64    `json:"id"`
	PublisherId int64    `json:"publisher_id"`
	CourseId    int64    `json:"course_id"`
	StarRating  uint32   `json:"star_rating"`
	Content     string   `json:"content"`
	Status      string   `json:"status"`
	Assessments []string `json:"assessments"` // 考核方式，支持多选
	Features    []string `json:"features"`    // 课程特点，支持多选
	Utime       int64    `json:"utime"`
	Ctime       int64    `json:"ctime"`
}

type EvaluationListCourseReq struct {
	CurEvaluationId int64 `form:"cur_evaluation_id"`
	Limit           int64 `form:"limit"`
}

type EvaluationListMineReq struct {
	CurEvaluationId int64  `form:"cur_evaluation_id"`
	Limit           int64  `form:"limit"`
	Status          string `form:"status"`
}

type EvaluationCountMineReq struct {
	Status string `form:"status"`
}
