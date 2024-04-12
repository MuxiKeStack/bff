package web

type QuestionPublishReq struct {
	Content string `json:"content"`
	Biz     string `json:"biz"`    // 平台资源类型，如course
	BizId   int64  `json:"biz_id"` // id
}

type RecommendationInviteesReq struct {
	CurUid int64 `form:"cur_uid"` // 第一页用 0 ，之后每次携带上一页的最后一个uid
	Limit  int64 `form:"limit"`
}

type InviteesVo struct {
	Uid      int64  `json:"uid"`
	Nickname string `json:"nickname"`
	Avatar   string `json:"avatar"`
}

type QuestionVo struct {
	Id           int64  `json:"id"`
	QuestionerId int64  `json:"questioner_id"` // 提问者用户id
	Biz          string `json:"biz"`           // 具体针对那种业务的提问，如 Course
	BizId        int64  `json:"biz_id"`        //
	Content      string `json:"content"`
}