package test2

import "github.com/peng225/silkroad/testdata/test1/test11"

type IF1 interface {
	Op1(a, b int) int
	OP2(a int) int
}

type IF2 interface {
	IF1
	test11.ST2
}
