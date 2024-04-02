package web

type LoginByCCNUReq struct {
	StudentId string `json:"student_id"`
	Password  string `json:"password"`
}
