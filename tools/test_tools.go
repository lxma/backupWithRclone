//go:build for_testing

package tools

import "github.com/lxma/golist/v2"

var gTestSetupFunctions = golist.New[TStandaloneFunction]()

func AddTestSetupFunc(setupFunc TStandaloneFunction) {
	gTestSetupFunctions.PushBack(setupFunc)
}

func TestSetup() {
	for f := range gTestSetupFunctions.Iter() {
		f()
	}
}
