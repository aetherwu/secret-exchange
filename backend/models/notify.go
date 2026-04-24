package models

import "time"

type FormID struct {
	User      string
	FormID    string
	CreatedAt time.Time
}
