package main

import (
	"fmt"

	"github.com/kcalixto/go-module/toolkit"
)

func main() {
	var tools toolkit.Tools

	path := "./non-existent-folder"

	err := tools.CreateDirIfNotExists(path)
	if err != nil {
		fmt.Println("failed to create dir: ", err.Error())
	}
}
