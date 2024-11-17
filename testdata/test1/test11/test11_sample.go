package test11

type ST1 struct {
	a int
}

type ST2 struct {
	st3 ST3
}

type ST3 struct {
	st4 []*ST4
}

type ST4 struct {
	b string
}

type ST5 struct {
	c int
	ST1
}

type ST6 struct {
	st7 map[string]*ST7
}

type ST7 struct {
	a uint8
}

func (st3 *ST3) Op1(a, b int) int {
	return a + b
}

func (st3 *ST3) Op2(a int) int {
	return -a
}
