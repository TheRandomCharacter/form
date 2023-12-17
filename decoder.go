package form

import (
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"strings"
)

type Decoder struct {
	delimiter        string
	cache            cacheT
	sliceDecodeFuncs map[reflect.Type]SliceDecodeFunc
	decodeFuncs      map[reflect.Type]DecodeFunc
	zeroinit         map[reflect.Type]bool
}

func NewDecoder() *Decoder {
	return &Decoder{
		delimiter:   ".",
		cache:       newCache(),
		decodeFuncs: predefinedDecodeFuncs(),
	}
}

var ErrInvalidDst = errors.New("destination must have named fields (struct, map, slice, array)")

func checkDstIsValid(t reflect.Type) error {
	//Need to also accept pointer to pointer to struct
	fmt.Println(t.Kind())
	if t.Kind() == reflect.Pointer {
		fmt.Print(t, " ")
		t = t.Elem()
		fmt.Println(t)
	}

	if t.Kind() == reflect.Struct || t.Kind() == reflect.Map || t.Kind() == reflect.Slice || t.Kind() == reflect.Array {
		return nil
	}

	return ErrInvalidDst
}

type srcNode struct {
	children map[string]*srcNode
	//value     []string
	converted any
}

func (n *srcNode) add(path []string, converted any) {
	if len(path) == 0 {
		n.converted = converted
	} else {
		if n.children == nil {
			n.children = make(map[string]*srcNode)
		}
		sub, ok := n.children[path[0]]
		if !ok {
			sub = &srcNode{}
			n.children[path[0]] = sub
		}
		sub.add(path[1:], converted)
	}
}

var ErrNoValuesMapped = errors.New("no valid mappings found (source empty if this is the only error)")
var ErrInvalidValue = errors.New("value conversion faild")
var ErrInvalidPath = errors.New("path is invalid")
var ErrTypeUnknown = errors.New("converter not found for type")

func (d *Decoder) findDecodeFunc(t reflect.Type) (df DecodeFunc, depth int, ok bool) {
	if df, ok = d.decodeFuncs[t]; ok {
		return
	}

	for t.Kind() == reflect.Pointer {
		t = t.Elem()
		depth++
		if df, ok = d.decodeFuncs[t]; ok {
			return
		}
	}

	return nil, 0, false
}

func indirectN(v reflect.Value, count int) reflect.Value {
	for ; count > 0; count-- {
		v = reflect.Indirect(v)
	}
	return v
}

func (d *Decoder) convertAggregate(
	valueMaker func() reflect.Value,
	l int,
	elem reflect.Type,
	values []string) (res any, warn []error, fatal error) {
	if dfunc, depth, ok := d.findDecodeFunc(elem); ok {
		v := valueMaker()
		valueSet := false
		for i := 0; i < l; i++ {
			if val, err := dfunc(values[i]); err == nil {
				if val != nil {
					v.Index(i).Set(indirectN(reflect.ValueOf(val), depth))
					valueSet = true
				}
			} else {
				warn = append(warn, err)
			}
		}
		if !valueSet {
			return nil, warn, ErrNoValuesMapped
		}
		return v.Interface(), warn, nil
	}
	return nil, nil, ErrTypeUnknown
}

func (d *Decoder) convert(t reflect.Type, values []string) (res any, warn []error, fatal error) {
	if sfunc, ok := d.sliceDecodeFuncs[t]; ok {
		v, e := sfunc(values)
		return v, nil, e
	}

	if t.Kind() == reflect.Slice {
		return d.convertAggregate(func() reflect.Value {
			return reflect.MakeSlice(t, len(values), len(values))
		}, len(values), t.Elem(), values)
	} else if t.Kind() == reflect.Array {
		return d.convertAggregate(func() reflect.Value {
			return reflect.New(t).Elem()
		}, len(values), t.Elem(), values)
	} else if values != nil && len(values) > 0 {
		if dfunc, ok := d.decodeFuncs[t]; ok {
			v, e := dfunc(values[len(values)-1])
			if e != nil {
				warn = append(warn, e)
				return nil, warn, ErrNoValuesMapped
			}
			return v, nil, nil
		}
	}

	return nil, nil, ErrTypeUnknown
}

func (d *Decoder) buildSourceTree(src map[string][]string, t reflect.Type) (root *srcNode, warns []error, fatal error) {
	if len(src) != 0 {
		root = &srcNode{}
		for key, value := range src {
			path := strings.Split(key, d.delimiter)
			element, ok := d.cache.elementType(t, path)
			if !ok {
				warns = append(warns, fmt.Errorf("\"%s\": %w", key, ErrInvalidPath))
				continue
			}
			converted, warn, err := d.convert(element, value)
			warns = append(warns, warn...)
			if err != nil {
				warns = append(warns, fmt.Errorf("\"%s\": \"%s\" %w %s", key, value, err, element))
			}
			if converted != nil {
				root.add(path, converted)
			}
		}
		if len(root.children) != 0 {
			return root, warns, nil
		} else {
			return nil, warns, ErrNoValuesMapped
		}
	} else {
		return nil, nil, ErrNoValuesMapped
	}
}

var defaultDecoder *Decoder = NewDecoder()

func chooseDecoder(custom ...*Decoder) *Decoder {
	if len(custom) == 0 {
		return defaultDecoder
	} else {
		return custom[0]
	}
}

func toIndices(ms map[string]*srcNode) (mi map[int]*srcNode, maxId int, warn []error) {
	maxId = -1
	mi = make(map[int]*srcNode)
	for name, val := range ms {
		id, err := strconv.Atoi(name)
		if err != nil {
			warn = append(warn, fmt.Errorf("bad slice index: \"%s\"", name))
			continue
		}
		if id < 0 {
			warn = append(warn, fmt.Errorf("slice index out of bounds: \"%s\"", name))
			continue
		}
		mi[id] = val
		maxId = max(maxId, id)
	}
	maxId++
	return
}

func (d *Decoder) pack(tree *srcNode, v reflect.Value) (warn []error, fatal error) {
	for v.Kind() == reflect.Pointer {
		if v.IsNil() {
			v.Set(reflect.New(v.Type().Elem()))
		}
		v = v.Elem()
	}

	var valueSet = false

	if tree.converted != nil {
		cv := reflect.ValueOf(tree.converted)
		v.Set(cv)
		valueSet = true
	}

	if len(tree.children) != 0 {
		switch v.Kind() {
		case reflect.Struct:
			info := d.cache.fetchCached(v.Type())
			for name, subtree := range tree.children {
				if field, ok := info[name]; ok {
					subwarn, subfatal := d.pack(subtree, v.FieldByIndex(field.index))
					warn = append(warn, subwarn...)
					if subfatal == nil {
						valueSet = true
					}
				}
			}
		case reflect.Map:
			if keyFunc, ok := d.decodeFuncs[v.Type().Key()]; ok {

				for keystring, subtree := range tree.children {
					key, err := keyFunc(keystring)
					if err != nil {
						warn = append(warn, fmt.Errorf("map key from \"%s\" string conversion failed (%w)", keystring, err))
						continue
					}
					elem := v.MapIndex(reflect.ValueOf(key))
					if !elem.IsValid() {
						elem = reflect.New(v.Type().Elem()).Elem()
					}
					subwarn, subfatal := d.pack(subtree, elem)
					warn = append(warn, subwarn...)
					if subfatal == nil {
						if v.IsNil() {
							v.Set(reflect.MakeMapWithSize(v.Type(), len(tree.children)))
						}
						v.SetMapIndex(reflect.ValueOf(key), elem)
						valueSet = true
					}
				}
			}
		case reflect.Slice:
			indexed, maxId, warns := toIndices(tree.children)
			warn = append(warn, warns...)
			if maxId != -1 {
				v.Set(reflect.MakeSlice(v.Type(), maxId, maxId))
				for i, subtree := range indexed {
					subwarn, subfatal := d.pack(subtree, v.Index(i))
					warn = append(warn, subwarn...)
					if subfatal == nil {
						valueSet = true
					}
				}
			}
		case reflect.Array:
			for name, subtree := range tree.children {
				idx, err := strconv.Atoi(name)
				if err != nil {
					warn = append(warn, fmt.Errorf("bad array index: \"%s\"", name))
					continue
				}
				if (idx < 0) || idx >= v.Len() {
					warn = append(warn, fmt.Errorf("array index out of bounds: \"%s\"", name))
					continue
				}
				subwarn, subfatal := d.pack(subtree, v.Index(idx))
				warn = append(warn, subwarn...)
				if subfatal == nil {
					valueSet = true
				}
			}
		}

	}

	if !valueSet {
		fatal = ErrNoValuesMapped
	}
	return
}

func Decode[T any](src map[string][]string, custom ...*Decoder) (res T, warn []error, e error) {
	dst := reflect.New(reflect.TypeOf(res)).Elem()
	if err := checkDstIsValid(dst.Type()); err != nil {
		return res, nil, err
	}
	var d *Decoder = chooseDecoder(custom...)
	tree, warns, err := d.buildSourceTree(src, reflect.TypeOf(res))
	if err != nil {
		return res, warns, err
	}
	morewarns, err := d.pack(tree, dst)
	warns = append(warns, morewarns...)
	return dst.Interface().(T), warns, err
}
