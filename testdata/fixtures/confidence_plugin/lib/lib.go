package lib

import "plugin"

func ExportedFunc() {}

func internalUse() {
	ExportedFunc()
	_, _ = plugin.Open("test.so")
}
