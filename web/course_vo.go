package web

type CourseListReq struct {
	Year string `json:"year"`
	Term string `json:"term"`
}

type ProfileCourseVo struct {
	Id        int64  `json:"id"`
	Name      string `json:"name"`
	Teacher   string `json:"teacher"`
	Evaluated bool   `json:"evaluated"`
	Year      string `json:"year"` // 学期，2018
	Term      string `json:"term"` // 学年，1/2/3
}

type SimplePublicCourseVo struct {
	Id       int64   `json:"id"`
	Name     string  `json:"name"`
	Teacher  string  `json:"teacher"`
	School   string  `json:"school"`
	Property string  `json:"type"`
	Credit   float64 `json:"credit"`
}

type PublicCourseVo struct {
	Id             int64            `json:"id"`
	Name           string           `json:"name"`
	Teacher        string           `json:"teacher"`
	School         string           `json:"school"`
	CompositeScore float64          `json:"composite_score"`
	RaterCount     int64            `json:"rater_count"`
	Property       string           `json:"type"`
	Credit         float64          `json:"credit"`
	Assessments    map[string]int64 `json:"assessments"` // 标签:数量
	Features       map[string]int64 `json:"features"`
	IsCollected    bool             `json:"is_collected"`  // 是否收藏了
	IsSubscribed   bool             `json:"is_subscribed"` // 是否上过这门课
}

type InviteUserToAnswerReq struct {
	Invitees []int64 `json:"invitees"`
}

type CourseQuestionPublishReq struct {
	Content string `json:"content"`
}

type CourseTagsVo struct {
	Assessments map[string]int64 `json:"assessments"` // 标签:数量
	Features    map[string]int64 `json:"features"`
}

type CourseListCollectionMineReq struct {
	CurCollectionId int64 `form:"cur_collection_id"`
	Limit           int64 `form:"limit"`
}

type CollectedCourseVo struct {
	Id             int64   `json:"id"`
	CourseId       int64   `json:"course_id"`
	Name           string  `json:"name"`
	Teacher        string  `json:"teacher"`
	School         string  `json:"school"`
	CompositeScore float64 `json:"composite_score"`
	Property       string  `json:"type"`
	Credit         float64 `json:"credit"`
	IsCollected    bool    `json:"is_collected"`
}

type CourseCollectReq struct {
	Collect bool `json:"collect"`
}
