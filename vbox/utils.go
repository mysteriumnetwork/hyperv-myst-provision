package vbox

type KVMap map[string]interface{}

func NewKVMap(i interface{}) KVMap {
	m, ok := i.(map[string]interface{})
	if !ok {
		return nil
	}
	return KVMap(m)
}
