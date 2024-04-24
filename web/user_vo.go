package web

type LoginByCCNUReq struct {
	StudentId string `json:"student_id"` // 学号
	Password  string `json:"password"`   // 密码
}

type UserEditReq struct {
	Avatar   string `json:"avatar"`
	Nickname string `json:"nickname"`
}

// UserProfileVo 自己的信息
type UserProfileVo struct {
	Id              int64  `json:"id"`
	StudentId       string `json:"studentId"`
	Avatar          string `json:"avatar"`
	Nickname        string `json:"nickname"`
	New             bool   `json:"new"` // 是否为新用户，新用户尚未编辑过个人信息
	GradeSignStatus string `json:"grade_sign_status"`
	Utime           int64  `json:"utime"`
	Ctime           int64  `json:"ctime"`
}

// UserPublicProfileVo 别人的信息
type UserPublicProfileVo struct {
	Id       int64  `json:"id"`
	Avatar   string `json:"avatar"`
	Nickname string `json:"nickname"`
}
