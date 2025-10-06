//go:build !darwin

package applications

import "fmt"

func listDarwin() ([]Application, error) {
	return nil, fmt.Errorf("listDarwin is only available on darwin")
}
