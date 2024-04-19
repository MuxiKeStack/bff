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

type PublicCourseVo struct {
	Id             int64            `json:"id"`
	Name           string           `json:"name"`
	Teacher        string           `json:"teacher"`
	School         string           `json:"school"`
	CompositeScore float64          `json:"composite_score"`
	Property       string           `json:"type"`
	Credit         float32          `json:"credit"`
	Assessments    map[string]int64 `json:"assessments"` // 标签:数量
	Features       map[string]int64 `json:"features"`
	IsCollected    bool             `json:"is_collected"`
	Grades         []Grade          `json:"grades"`
}

type Grade struct {
	Regular float32 `json:"regular"`
	Final   float32 `json:"final"`
	Total   float32 `json:"total"`
	Year    string  `json:"year"`
	Term    string  `json:"term"`
}

type GradeVo struct {
	Regular float32 `json:"regular"`
	Final   float32 `json:"final"`
	Total   float32 `json:"total"`
	Year    string  `json:"year"`
	Term    string  `json:"term"`
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
	CollectionId   int64   `json:"collection_id"`
	Name           string  `json:"name"`
	Teacher        string  `json:"teacher"`
	School         string  `json:"school"`
	CompositeScore float64 `json:"composite_score"`
	Property       string  `json:"type"`
	Credit         float32 `json:"credit"`
	IsCollected    bool    `json:"is_collected"`
}
