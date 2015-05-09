package goweb

import (
	"log"
	"os"
)

var (
	Debug = true
)

const (
	ActionPrefix     = "Action"
)

var Log = log.New(os.Stdout, "[GWLOG]", log.Ldate|log.Lmicroseconds|log.Lshortfile)
var Err = log.New(os.Stderr, "[GWERR]", log.Ldate|log.Lmicroseconds|log.Lshortfile)

const (
	LifeTypeError = iota
	LifeTypeStateless
	LifeTypeStandalone
	LifeTypeStateful
)

type LifeType int
