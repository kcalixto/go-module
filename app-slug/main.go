package main

import (
	"fmt"

	"github.com/kcalixto/go-module/toolkit"
)

func main() {
	var tools toolkit.Tools

	s := "Hello, Babyyy!!"

	slug, err := tools.Slugify(s)
	if err != nil {
		fmt.Println("failed to slugify: ", err.Error())
	}

	fmt.Println("new slug: ", slug)
}
