package main

import "C"

import (
	"gowsfe/wsafip"
)

//export GetUltimo
func GetUltimo() {
	wsafip.GetUltimo()
}

func main() {
	GetUltimo()
}
