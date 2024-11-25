package t11

import (
	"time"

	ioalias "io"

	"github.com/peng225/silkroad/tdata/t2"
)

var _ t2.IF1 = (*ST3)(nil)

type ST1 struct {
	a int
}

type ST2 struct {
	st3 ST3
	t   time.Duration
}

type ST3 struct {
	st4   []*ST4
	st4_2 []ST4
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
	a   uint8
	st8 *ST8
}

type ST8 struct {
	v uint8
	w ioalias.Reader
}

func (s *ST3) Op1(a, b int) int {
	return a + b
}

func (s *ST3) Op2(a int) int {
	return -a
}

func (s *ST3) Op3(o ...t2.ST200) {
	return
}
