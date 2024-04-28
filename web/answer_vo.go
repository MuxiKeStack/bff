package web

type AnswerPublishReq struct {
	QuestionId int64  `json:"question_id"`
	Content    string `json:"content"`
}

type AnswerListReq struct {
	CurAnswerId int64 `form:"cur_answer_id"`
	Limit       int64 `form:"limit"`
}

type AnswerVo struct {
	Id                int64  `json:"id"`
	PublisherId       int64  `json:"publisher_id"`
	QuestionId        int64  `json:"question_id"`
	Content           string `json:"content"`
	Stance            int32  `json:"stance"`
	TotalSupportCount int64  `json:"total_support_count"`
	TotalOpposeCount  int64  `json:"total_oppose_count"`
	TotalCommentCount int64  `json:"total_comment_count"`
	Utime             int64  `json:"utime"`
	Ctime             int64  `json:"ctime"`
}
