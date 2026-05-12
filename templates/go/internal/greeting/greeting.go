package greeting

import (
	"fmt"
	"strings"
)

type Input struct {
	Name string
}

func Create(input Input) (string, error) {
	name := strings.TrimSpace(input.Name)
	if name == "" {
		return "", fmt.Errorf("name must not be empty")
	}

	return fmt.Sprintf("Hello, %s.", name), nil
}
