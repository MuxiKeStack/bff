package web

type EvaluationPublishReq struct {
	CourseId    int64    `json:"course_id"`
	StarRating  uint8    `json:"star_rating"` // 1，2，3，4，5
	Content     string   `json:"content"`     // 评价的内容
	Assessments []string `json:"assessments"` // 考核方式，支持多选
	Features    []string `json:"features"`    // 课程特点，支持多选
	Anonymous   bool     `json:"anonymous"`   // 是否匿名提交
}
