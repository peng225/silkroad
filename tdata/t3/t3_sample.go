package t3

import "github.com/peng225/silkroad/tdata/t1/t11"

type AliasForST100 ST100
type AliasForMapST100 map[string]ST100
type AliasForSliceST100 []ST100
type AliasForArrayST100 [2]ST100
type AliasForStarST100 *ST100
type AliasForChanInt chan AliasForInt
type AliasForInt int
type AliasForAny any
type AliasForEmptyStruct struct{}
type AliasForFunc func(*t11.ST1) *t11.ST2

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

type ST103 struct {
	h chan AliasForInt
	i <-chan AliasForInt
	j chan<- AliasForInt
	k chan int
}

type ST104[T comparable] struct {
	a complex128
	b complex64
}

type ST105 struct {
	error
	f func(*t11.ST1) *t11.ST2
	g ST101[t11.ST3]
	h ST104[int]
}

type ST106 struct {
	f AliasForFunc
	l AliasForChanInt
}
