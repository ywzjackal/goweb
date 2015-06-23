package goweb

import (
	"log"
	"os"
)

var (
	Debug = true
)

var Log = log.New(os.Stdout, "[GWLOG]", log.Ldate|log.Lmicroseconds|log.Lshortfile)
var Err = log.New(os.Stderr, "[GWERR]", log.Ldate|log.Lmicroseconds|log.Lshortfile)

type LifeType int

const (
	LifeTypeError LifeType = iota
	LifeTypeStateless
	LifeTypeStandalone
	LifeTypeStateful
)

var LifeTypeName = []string{
	LifeTypeError:      "LifeTypeError",
	LifeTypeStateless:  "LifeTypeStateless",
	LifeTypeStandalone: "LifeTypeStandalone",
	LifeTypeStateful:   "LifeTypeStateful",
}
