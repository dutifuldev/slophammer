package main

import (
	"fmt"
	"os"

	"example.com/slophammer-go-template/internal/greeting"
)

func main() {
	message, err := greeting.Create(greeting.Input{Name: "world"})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	fmt.Println(message)
}
