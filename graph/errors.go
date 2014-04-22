package graph

import "errors"

var ErrSymbolNotExist = errors.New("symbol does not exist")

func IsNotExist(err error) bool {
	return err == ErrSymbolNotExist
}
