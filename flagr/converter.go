package flagr

import (
	"fmt"
	"strconv"
)

type Converter func(string) (interface{}, error)

// A set of converters for string values.
var converters = map[string]Converter{
	"int": toInt,
}

func AddConverter(n string, c Converter) error {
	if _, ok := converters[n]; ok {
		return fmt.Errorf("converter name %q is reserved", n)
	}
	converters[n] = c
	return nil
}

func GetConverter(n string) (Converter, error) {
	c, ok := converters[n]
	if !ok {
		return nil, fmt.Errorf("converter name %q is not found", n)
	}
	return c, nil
}

func toInt(v string) (interface{}, error) {
	i, err := strconv.Atoi(v)
	if err != nil {
		return 0, err
	}
	return i, nil
}
