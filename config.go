package goweb

import (
	"log"
	"os"
)

var (
	Debug = true
)

var Log = log.New(os.Stdout, "[GWLOG]", log.Ldate|log.Lmicroseconds|log.Lshortfile)
var Err = log.New(os.Stderr, "[GWERR]", log.Ldate|log.Lmicroseconds|log.Llongfile)

type LifeType int

const (
	LifeTypeError LifeType = iota
	LifeTypeStateless
	LifeTypeStandalone
	LifeTypeStateful
)

func (l LifeType) String() string {
	switch l {
	case LifeTypeStandalone:
		return "LifeTypeStandalone"
	case LifeTypeStateless:
		return "LifeTypeStateless"
	case LifeTypeStateful:
		return "LifeTypeStateful"
	default:
		return "LifeTypeError"
	}
}
