package web

type LoginByCCNUReq struct {
	StudentId string `json:"student_id"` // 学号
	Password  string `json:"password"`   // 密码
}

type UserEditReq struct {
	Avatar   string `json:"avatar"`
	Nickname string `json:"nickname"`
}

type UserVo struct {
	Id        int64
	StudentId string
	Avatar    string
	Nickname  string
	New       bool
	Utime     int64
	Ctime     int64
}
