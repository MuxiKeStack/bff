package web

type CommentPublishReq struct {
	Biz      string `json:"biz"`
	BizId    int64  `json:"biz_id"`
	Content  string `json:"content"`
	RootId   int64  `json:"root_id"`
	ParentId int64  `json:"parent_id"`
}

type CommentListReq struct {
	Biz          string `form:"biz"`
	BizId        int64  `form:"biz_id"`
	CurCommentId int64  `form:"cur_comment_id"`
	Limit        int64  `form:"limit"`
}

type CommentVo struct {
	Id              int64  `json:"id"`
	CommentatorId   int64  `json:"commentator_id"`
	Biz             string `json:"biz"`
	BizId           int64  `json:"biz_id"`
	Content         string `json:"content"`
	RootCommentId   int64  `json:"root_comment_id"`
	ParentCommentId int64  `json:"parent_comment_id"`
	ReplyToUid      int64  `json:"reply_to_uid"`
	Utime           int64  `json:"utime"`
	Ctime           int64  `json:"ctime"`
}

type CommentListReliesReq struct {
	RootId       int64 `form:"root_id"`
	CurCommentId int64 `form:"cur_comment_id"`
	Limit        int64 `form:"limit"`
}

type CommentCountReq struct {
	Biz   string `form:"biz"`
	BizId int64  `form:"biz_id"`
}
