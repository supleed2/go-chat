package common

import "time"

type SMsg struct {
	Tim time.Time
	Id  string
	Msg string
}

type CMsgT int

const (
	Sudo CMsgT = iota
	Echo
	Mv
	Ls
	Cd
	Who
)

type CMsg struct {
	Typ CMsgT
	Msg string
}
