package main

import (
	"github.com/qaqcatz/nanoshlib"
	"os"
)

func pathExists(path string) bool {
	_, err := os.Stat(path)
	if err == nil {
		return true
	}
	if os.IsNotExist(err) {
		return false
	}
	panic("[pathExists]os.Stat error: " + err.Error())
}

func linuxCP(src string, dest string) {
	_, errStream, err := nanoshlib.Exec("cp " + src + " " + dest, -1)
	if err != nil {
		panic("[linuxCP]cp error: " + err.Error() + ": " + errStream)
	}
}