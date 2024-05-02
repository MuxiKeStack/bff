package web

type GetFeedEventsListReq struct {
	LastTime int64 `form:"last_time"`
	Limit    int64 `form:"limit"`
}
