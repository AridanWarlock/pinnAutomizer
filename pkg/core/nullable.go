package core

type Nullable[T any] struct {
	Value *T
	Set   bool
}
