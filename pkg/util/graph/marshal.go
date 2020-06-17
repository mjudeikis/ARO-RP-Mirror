package graph

import (
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/openshift/installer/pkg/asset"
)

func (g Graph) MarshalJSON() ([]byte, error) {
	m := map[string]asset.Asset{}
	for t, a := range g {
		m[t.String()] = a
	}
	return json.Marshal(m)
}

func (g *Graph) UnmarshalJSON(b []byte) error {
	if *g == nil {
		*g = Graph{}
	}

	var m map[string]json.RawMessage
	err := json.Unmarshal(b, &m)
	if err != nil {
		return err
	}

	for n, b := range m {
		t, found := registeredTypes[n]
		if !found {
			// TODO : return fmt.Errorf("unregistered type %q", n)
			fmt.Printf("unregistered type %q", n)
			continue
		}

		a := reflect.New(reflect.TypeOf(t).Elem()).Interface().(asset.Asset)
		err = json.Unmarshal(b, a)
		if err != nil {
			return err
		}

		(*g)[reflect.TypeOf(a)] = a
	}

	return nil
}
