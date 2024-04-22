package search

type SearchReq struct {
	// 需要指定biz，如course，以后拓展可以增加不指定biz的逻辑
	Biz            string `form:"biz"`
	Keyword        string `form:"keyword"`
	SearchLocation string `form:"search_location"` // 可以是 "home" 或 "favorites"
}

type DeleteHistoryReq struct {
	SearchLocation string  `json:"search_location"` // 可以是 "home" 或 "favorites"
	RemoveAll      bool    `json:"remove_all"`
	HistoryIds     []int64 `json:"history_ids"`
}

type GetHistoryReq struct {
	SearchLocation string `form:"search_location"` // 可以是 "home" 或 "favorites"
}

type HistoryVo struct {
	Id      int64  `json:"id"`
	Keyword string `json:"keyword"`
}
