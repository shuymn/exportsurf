package lib

import "reflect"

type ReflectedType struct{}

//export CgoExportedFunc
func CgoExportedFunc() {}

func NormalFunc() {}

func ExternallyUsedFunc() {}

func internalUse() {
	_ = ReflectedType{}
	_ = reflect.TypeOf(ReflectedType{})
	CgoExportedFunc()
	NormalFunc()
	ExternallyUsedFunc()
	LinkedFunc()
}
