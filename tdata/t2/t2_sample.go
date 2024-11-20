package t2

type IF1 interface {
	Op1(a, b int) int
	Op2(a int) int
}

type IF2 interface {
	IF1
}
