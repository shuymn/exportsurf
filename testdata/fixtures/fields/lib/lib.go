package lib

type MyStruct struct {
	InternalField string
	ExternalField string
	EmbeddedType
	TaggedField int `json:"tagged"`
}

type EmbeddedType struct{}

type GenericStruct[T any] struct {
	Value T
}

type unexported struct {
	PublicField string
}

func internalUse() {
	s := MyStruct{}
	_ = s.InternalField
	_ = s.ExternalField
	_ = s.EmbeddedType
	_ = s.TaggedField

	g := GenericStruct[int]{}
	_ = g.Value

	u := unexported{}
	_ = u.PublicField
}
