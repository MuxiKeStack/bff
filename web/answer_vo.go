package web

type AnswerPublishReq struct {
	QuestionId int64  `json:"question_id"`
	Content    string `json:"content"`
}

type AnswerListForQuestionReq struct {
	CurAnswerId int64 `json:"cur_answer_id"`
	Limit       int64 `json:"limit"`
}
