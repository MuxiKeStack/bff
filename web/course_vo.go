package web

type CourseListReq struct {
	Year string `json:"year"`
	Term string `json:"term"`
}

type ProfileCourseVo struct {
	StudentId string `json:"student_id"`
	CourseId  string `json:"course_id"`
	Name      string `json:"name"`
	Teacher   string `json:"teacher"`
	Evaluated bool   `json:"evaluated"`
	Year      string `json:"year"` // 学期，2018
	Term      string `json:"term"` // 学年，1/2/3
}
