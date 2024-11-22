package t3

type AliasForST100 ST100
type AliasForMapST100 map[string]ST100
type AliasForSliceST100 []ST100
type AliasForArrayST100 [2]ST100
type AliasForStarST100 *ST100
type AliasForInt int
type AliasForAny any
type AliasForEmptyStruct struct{}

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
	q AliasForAny
	r AliasForEmptyStruct
	s AliasForStarST100
	t AliasForArrayST100
	u AliasForSliceST100
}
