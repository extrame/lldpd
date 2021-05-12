package lldpd

import (
	"log"
)

type Logger interface {
	Info(...interface{})
	Error(...interface{})
	Errorf(string, ...interface{})
}

type NormalLoger struct {
}

func (n *NormalLoger) Info(args ...interface{}) {
	log.Print(args)
}

func (n *NormalLoger) Error(args ...interface{}) {
	log.Print(args)
}

func (n *NormalLoger) Errorf(fmt string, args ...interface{}) {
	log.Printf(fmt, args)
}
