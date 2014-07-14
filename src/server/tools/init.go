package tools

import "unsafe"

//type for implementation of Cacheable interface
type RawByteString struct {
	value []byte
	key string
}

// Following function implements interface Key() and returns key of the value
func (container *RawByteString) Key() string {
	return container.key
}

// Following function implements interface Size() and returns amount of bytes
func (container *RawByteString) Size() int {
	return len(container.value)
}

//interface for different types of protocols
type Protocol interface {
	// to think about implementation
}

func In(element string, collection []string) bool{
	for index, value := range collection {
		if element == value { return true }
	}
	return false
}
