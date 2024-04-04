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
	Id        int64  `json:"id,omitempty"`
	StudentId string `json:"studentId,omitempty"`
	Avatar    string `json:"avatar,omitempty"`
	Nickname  string `json:"nickname,omitempty"`
	New       bool   `json:"new,omitempty"` // 是否为新用户，新用户尚未编辑过个人信息
	Utime     int64  `json:"utime,omitempty"`
	Ctime     int64  `json:"ctime,omitempty"`
}

// UserPublicProfileVo 别人的信息
type UserPublicProfileVo struct {
	Id       int64  `json:"id,omitempty"`
	Avatar   string `json:"avatar,omitempty"`
	Nickname string `json:"nickname,omitempty"`
}
