package domain

import "io"

type TaskFile struct {
	Name string
	File io.ReadCloser
}
