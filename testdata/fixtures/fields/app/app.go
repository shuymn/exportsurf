package app

import "github.com/shuymn/exportsurf/testdata/fixtures/fields/lib"

func use() {
	s := lib.MyStruct{}
	_ = s.ExternalField
	var _ lib.GenericStruct[int]
	var _ lib.EmbeddedType
}
