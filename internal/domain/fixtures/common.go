package fixtures

type mod[T any] func(entity *T)

func fixture[T any](entity T, mods []mod[T]) T {
	for _, m := range mods {
		m(&entity)
	}
	return entity
}
