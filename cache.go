package form

import (
	"reflect"
	"strings"
)

type fieldInfo struct {
	Type  reflect.Type
	index []int
}

type structInfo map[string]fieldInfo

type cacheT struct {
	data           map[reflect.Type]structInfo
	tagKey         string
	tagDelimiter   string
	tagValueIgnore string
}

func newCache() cacheT {
	return cacheT{
		tagKey:         "form",
		tagDelimiter:   ",",
		tagValueIgnore: "-",
		data:           map[reflect.Type]structInfo{},
	}
}

func (c *cacheT) elementType(t reflect.Type, path []string) (reflect.Type, bool) {
	for _, leg := range path {
		for t.Kind() == reflect.Pointer {
			t = t.Elem()
		}

		if t.Kind() == reflect.Struct {
			info := c.fetchCached(t)
			if field, ok := info[leg]; ok {
				t = t.FieldByIndex(field.index).Type
			} else {
				return t, false
			}
			continue
		}

		if t.Kind() == reflect.Map || t.Kind() == reflect.Array || t.Kind() == reflect.Slice {
			t = t.Elem()
		} else {
			return t, false
		}
	}
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	return t, true
}

// t must be of struct type
func (c *cacheT) appendCache(t reflect.Type) structInfo {
	var info structInfo = make(structInfo)
	for _, field := range reflect.VisibleFields(t) {
		if field.IsExported() {
			key := field.Name
			if tag, ok := field.Tag.Lookup(c.tagKey); ok && len(tag) != 0 {
				if tag == c.tagValueIgnore {
					continue
				}
				vals := strings.Split(tag, c.tagDelimiter)
				if len(vals[0]) != 0 {
					key = vals[0]
				}
			}
			info[key] = fieldInfo{
				field.Type,
				field.Index,
			}
		}
	}
	c.data[t] = info
	return info
}

// t must be of struct type
func (c *cacheT) fetchCached(t reflect.Type) (info structInfo) {
	if info, ok := c.data[t]; ok {
		return info
	} else {
		return c.appendCache(t)
	}
}
