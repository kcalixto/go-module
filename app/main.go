package main

import (
	"fmt"
	"github.com/kcalixto/go-module/toolkit"
)

func main() {
	var tools toolkit.Tools

	s := tools.RandomString(10)
	fmt.Println("Random String: ", s)
}
