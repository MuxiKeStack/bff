package web

type GetStaticReq struct {
	StaticName string `form:"static_name"`
}

type StaticVo struct {
	Content string `json:"content"`
}

type SaveStaticReq struct {
	Name    string `json:"name"`
	Content string `json:"content"`
}

type SaveStaticByFileReq struct {
	Name string `form:"name"`
}
