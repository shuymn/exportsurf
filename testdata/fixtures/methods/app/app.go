package app

import "github.com/shuymn/exportsurf/testdata/fixtures/methods/lib"

func use() {
	t := lib.MyType{}
	_ = t.ExternallyUsed()
	var _ lib.Container[int]
}
