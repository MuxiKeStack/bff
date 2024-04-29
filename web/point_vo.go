package web

type PointInfoVo struct {
	Points          int64 `json:"points"`
	NextLevelPoints int64 `json:"next_level_points"`
	Level           int64 `json:"level"`
}
