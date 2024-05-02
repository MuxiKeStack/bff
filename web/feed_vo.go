package web

type GetFeedEventsListReq struct {
	LastTime  int64  `form:"last_time"` // 获取到的上一条消息的发生事件ctime
	Direction string `form:"direction"` // 查询方向 before 或 after last_time
	Limit     int64  `form:"limit"`
}
