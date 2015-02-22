package eventstore

import (
	"fmt"
	"reflect"
)

type TypeMap map[string]reflect.Type

func (typeMap TypeMap) RegisterType(event Event) {
	typeMap[event.Tag()] = reflect.TypeOf(event)
}

func (typeMap TypeMap) EventForType(typename string) Event {
	typ, ok := typeMap[typename]
	if !ok {
		panic(fmt.Errorf("type %q not registered.", typename))
	}

	return reflect.New(typ.Elem()).Interface().(Event)
}
