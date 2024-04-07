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
	Id       int64   `json:"id"`
	Name     string  `json:"name"`
	Teacher  string  `json:"teacher"`
	School   string  `json:"school"`
	Property string  `json:"type"`
	Credit   float32 `json:"credit"`
}

type GradeVo struct {
	Regular float32 `json:"regular"`
	Final   float32 `json:"final"`
	Total   float32 `json:"total"`
	Year    string  `json:"year"`
	Term    string  `json:"term"`
}
