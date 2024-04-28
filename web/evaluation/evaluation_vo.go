package evaluation

import "github.com/MuxiKeStack/bff/web"

type SaveReq struct {
	Id          int64    `json:"id"`
	CourseId    int64    `json:"course_id"`
	StarRating  uint8    `json:"star_rating"` // 1，2，3，4，5
	Content     string   `json:"content"`     // 评价的内容
	Assessments []string `json:"assessments"` // 考核方式，支持多选
	Features    []string `json:"features"`    // 课程特点，支持多选
	Status      string   `json:"status"`      // 可见性：Public/Private
}

type UpdateStatusReq struct {
	Status string `json:"status"`
}

type ListRecentReq struct {
	CurEvaluationId int64  `form:"cur_evaluation_id"`
	Limit           int64  `form:"limit"`
	Property        string `form:"property"`
}

type EvaluationVo struct {
	Id                int64    `json:"id"`
	PublisherId       int64    `json:"publisher_id"`
	CourseId          int64    `json:"course_id"`
	StarRating        uint32   `json:"star_rating"`
	Content           string   `json:"content"`
	Status            string   `json:"status"`
	Assessments       []string `json:"assessments"` // 考核方式，支持多选
	Features          []string `json:"features"`    // 课程特点，支持多选
	Stance            int32    `json:"stance"`      // 1支持，0无，-1反对
	TotalSupportCount int64    `json:"total_support_count"`
	TotalOpposeCount  int64    `json:"total_oppose_count"`
	TotalCommentCount int64    `json:"total_comment_count"`
	Utime             int64    `json:"utime"`
	Ctime             int64    `json:"ctime"`
}

type ListCourseReq struct {
	CurEvaluationId int64 `form:"cur_evaluation_id"`
	Limit           int64 `form:"limit"`
}

type ListMineReq struct {
	CurEvaluationId int64  `form:"cur_evaluation_id"`
	Limit           int64  `form:"limit"`
	Status          string `form:"status"`
}

type CountMineReq struct {
	Status string `form:"status"`
}

type EndorseReq = web.EndorseReq
