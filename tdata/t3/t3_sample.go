package t3

type AliasForST100 ST100
type AliasForMapST100 map[string]ST100
type AliasForInt int

type ST100 struct {
	a any
	b interface{}
	c struct{}
}

type ST101[T any] struct {
	v []T
	w *AliasForST100
	x AliasForInt
}

type ST102 struct {
	p AliasForMapST100
}
