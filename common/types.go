package common

import "time"

const Version = "0.2.11"

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
