package client

import "fmt"

func GenerateId(id int) string {
	return fmt.Sprintf("%s:%d", "robot", id)
}
