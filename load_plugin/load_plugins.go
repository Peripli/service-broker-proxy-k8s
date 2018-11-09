package load_plugin

import (
	"fmt"
	"github.com/Peripli/service-manager/pkg/web"
	"plugin"
	"reflect"
	"unsafe"
)

func LoadPlugins(pluginList []string, api *web.API) error {
	for _, p := range pluginList {
		plugIn, err := plugin.Open(p)
		if err != nil {
			return fmt.Errorf("Unable to load plugin %s: %v", p, err)
		}
		symbol, err := plugIn.Lookup("Init")
		if err != nil {
			return fmt.Errorf("Unable to find symbol: 'Init' in plugin %s", p)
		}
		fmt.Printf("%v\n", reflect.TypeOf(symbol))

		init, ok := symbol.(func(unsafe.Pointer) error)

		if !ok {
			return fmt.Errorf("Symbol 'Init' in plugin %s is not of expected type 'func(*web.API) error'", p)
		}
		err = init(unsafe.Pointer(api))
		if err != nil {
			return fmt.Errorf("Error during initialization of plugin %s: %v", p, err)
		}
	}
	return nil
}
