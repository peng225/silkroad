package t2

type ST200 struct {
	a int
}

type IF1 interface {
	Op1(a, b int) int
	Op2(a int) int
	Op3(o ...ST200)
}

type IF2 interface {
	IF1
}
