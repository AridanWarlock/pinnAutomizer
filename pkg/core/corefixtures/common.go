package corefixtures

type Mod[T any] func(entity *T)

func Fixture[T any](entity T, mods []Mod[T]) T {
	for _, m := range mods {
		m(&entity)
	}
	return entity
}
