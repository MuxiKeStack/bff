package search

type SearchReq struct {
	// 需要指定biz，如course，以后拓展可以增加不指定biz的逻辑
	Biz            string `form:"biz"`
	Keyword        string `form:"keyword"`
	SearchLocation string `form:"search_location"` // 可以是 "Home" 或 "Collections"
}

type DeleteHistoryReq struct {
	SearchLocation   string  `json:"search_location"` // 可以是 "Home" 或 "Collections"
	RemoveAll        bool    `json:"remove_all"`
	RemoveHistoryIds []int64 `json:"remove_history_ids"`
}

type GetHistoryReq struct {
	SearchLocation string `form:"search_location"` // 可以是 "Home" 或 "Collections"
}

type HistoryVo struct {
	Id      int64  `json:"id"`
	Keyword string `json:"keyword"`
}
