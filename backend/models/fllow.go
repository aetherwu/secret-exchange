package models

import "time"

type Follows struct {
	OpenID    string
	Follows   []Follow // 关注的人
	Followers []Follow // 粉丝
}

type Follow struct {
	OpenID    string
	CreatedAt time.Time
}
