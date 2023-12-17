package form

import (
	"fmt"
	"net/url"
	"testing"
	"time"
)

func errorChecker(t *testing.T) func(error) {
	t.Helper()
	return func(e error) {
		t.Helper()
		if e != nil {
			t.Fatal(e)
		}
	}
}

func assertTrue(t *testing.T, Msg ...any) func(bool) {
	t.Helper()
	return func(b bool) {
		t.Helper()
		if !b {
			t.Fatal(Msg...)
		}
	}
}

func TestDecodeStruct(t *testing.T) {
	type Container struct {
		FieldA string
		FieldB int
	}

	sval := "Hello world"
	ival := 42

	values := map[string][]string{
		"FieldA": {sval},
		"FieldB": {fmt.Sprint(ival)},
	}

	c, warn, err := Decode[Container](values)

	if err != nil || warn != nil {
		t.Fatal(warn, err)
	}

	if c.FieldA != sval || c.FieldB != ival {
		t.Fatalf("Decoded values don't match source:\"%s\"->\"%s\", %d -> %d",
			sval, c.FieldA, ival, c.FieldB)
	}

	cp, warn, err := Decode[*Container](values)

	if err != nil || warn != nil {
		t.Fatal(warn, err)
	}

	if cp.FieldA != sval || cp.FieldB != ival {
		t.Fatalf("Decoded values don't match source:\"%s\"->\"%s\", %d -> %d",
			sval, cp.FieldA, ival, cp.FieldB)
	}
}

func TestDecodeStructEmbedded(t *testing.T) {
	type E struct {
		I int
		S string
	}
	type R struct {
		E
		S string
	}
	i := 42
	s := "Hello world!"
	ss := "Shadowed"
	values := map[string][]string{
		"I":   {fmt.Sprint(i)},
		"S":   {s},
		"E.S": {ss},
	}

	r, warn, err := Decode[R](values)
	if err != nil || warn != nil {
		t.Fatal(warn, err)
	}

	assertTrue(t, "embedded integer is not set")(r.I == i)
	assertTrue(t, "string is not set")(r.S == s)
	assertTrue(t, "shadowed string is not set")(r.E.S == ss)
}

func TestDecodeMap(t *testing.T) {
	values := map[string][]string{
		"1": {"True"},
		"0": {"False"},
	}

	res, warn, err := Decode[map[int]bool](values)

	if err != nil || warn != nil {
		t.Fatal(warn, err)
	}
	assertTrue(t, "mapping is wrong")(len(res) == 2 && res[1] && !res[0])
}

func TestDecodeSliceContainer(t *testing.T) {
	values := map[string][]string{
		"0": {"42"},
		"1": {"42"},
		"3": {"42"},
		"4": {"42"},
	}

	res, warn, err := Decode[[]int](values)
	if err != nil || warn != nil {
		t.Fatal(warn, err)
	}
	assertTrue(t, "mapping is wrong")(len(res) == 5 && res[0] == 42 && res[2] == 0 && res[4] == 42)
}

func TestDecodeSliceValue(t *testing.T) {
	type Container struct {
		Slice []int
	}
	values := map[string][]string{
		"Slice": {"2", "4", "6"},
	}

	res, warn, err := Decode[Container](values)
	if err != nil || warn != nil {
		t.Fatal(warn, err)
	}
	s := res.Slice
	assertTrue(t, "mapping is wrong")(len(s) == 3 && s[0] == 2 && s[1] == 4 && s[2] == 6)
}

func TestDecodeArrayContainer(t *testing.T) {
	values := map[string][]string{
		"0": {"42"},
		"1": {"42"},
		"3": {"42"},
		"4": {"42"},
	}

	res, warn, err := Decode[[5]int](values)
	if err != nil || warn != nil {
		t.Fatal(warn, err)
	}
	assertTrue(t, "mapping is wrong")(len(res) == 5 && res[0] == 42 && res[2] == 0 && res[4] == 42)

	resp, warn, err := Decode[[5]int](values)
	if err != nil || warn != nil {
		t.Fatal(warn, err)
	}
	assertTrue(t, "mapping is wrong")(len(resp) == 5 && resp[0] == 42 && resp[2] == 0 && resp[4] == 42)

}

func TestDecodeArrayValue(t *testing.T) {
	type Container struct {
		Array [3]int
	}
	values := map[string][]string{
		"Array": {"2", "4", "6"},
	}

	res, warn, err := Decode[Container](values)
	if err != nil || warn != nil {
		t.Fatal(warn, err)
	}
	a := res.Array
	assertTrue(t, "mapping is wrong")(len(a) == 3 && a[0] == 2 && a[1] == 4 && a[2] == 6)

	type ContainerP struct {
		Pointer *[3]int
	}
	values = map[string][]string{
		"Pointer": {"8", "10", "12"},
	}

	resp, warn, err := Decode[ContainerP](values)
	if err != nil || warn != nil {
		t.Fatal(warn, err)
	}

	ap := resp.Pointer
	assertTrue(t, "mapping is wrong")(len(ap) == 3 && ap[0] == 8 && ap[1] == 10 && ap[2] == 12)
}

func TestNilPreservation(t *testing.T) {
	type C struct {
		Number       int
		NumberP      *int
		NilP         *int
		NilPKeyed    *int
		NilPKeyedNil *int
	}
	i := 42
	values := map[string][]string{
		"Number":       {fmt.Sprint(i)},
		"NumberP":      {fmt.Sprint(i)},
		"NilPKeyed":    {""},
		"NilPKeyedNil": {},
	}
	r, warn, err := Decode[C](values)
	if err != nil || warn != nil {
		t.Fatal(warn, err)
	}

	assertTrue(t, "value doesn't map to pointer")(*(r.NumberP) == i)
	assertTrue(t, "nullptr is zeroed without a value")(r.NilP == nil)
	assertTrue(t, "empty value inits nullptr")(r.NilPKeyed == nil)
	assertTrue(t, "nil value inits nullptr")(r.NilPKeyedNil == nil)
}

func TestIntermediaryNullptrPreservation(t *testing.T) {
	type A struct {
		A1 int
		A2 int
	}

	type C struct {
		C1 *A
		C2 *A
	}

	i := 42
	values := map[string][]string{
		"K1.C1.A1":             {fmt.Sprint(i)},
		"K2.C2.A1.Nonexistent": {"value"},
	}

	r, _, err := Decode[map[string]*C](values)
	if err != nil {
		t.Fatal(err)
	}

	if v, ok := r["K1"]; ok {
		assertTrue(t, "value not decoded")(v.C1.A1 == i)
		assertTrue(t, "nullptr initialized without value")(v.C2 == nil)
	} else {
		t.Error("value is not added to map")
	}

	if _, ok := r["K2"]; ok {
		t.Error("element added to map without value")
	}

}

func TestCustomName(t *testing.T) {
	type C struct {
		Hello string `form:"h"`
		World string `form:"w"`
	}

	h := "hello"
	w := "world"

	values := map[string][]string{
		"h": {h},
		"w": {w},
	}

	r, _, err := Decode[C](values)
	if err != nil {
		t.Fatal(err)
	}

	assertTrue(t, "values don't match")(r.Hello == h && r.World == w)
}

func TestIgnoreField(t *testing.T) {
	type C struct {
		I int `form:"-"`
		J int
	}

	i := 42
	values := map[string][]string{
		"I": {fmt.Sprint(i)},
		"J": {fmt.Sprint(i)},
	}

	r, _, err := Decode[C](values)
	if err != nil {
		t.Fatal(err)
	}

	assertTrue(t, "value not ignored")(r.I == 0 && r.J == i)
}

func TestDecodeUrlValue(t *testing.T) {
	i := 42
	values := url.Values{
		"I": {fmt.Sprint(i)},
	}
	type C struct {
		I int
	}

	r, warn, err := Decode[C](values)
	if err != nil || warn != nil {
		t.Fatal(warn, err)
	}

	assertTrue(t, "url.Values must be valid source")(r.I == i)
}

func TestDefaultConverters(t *testing.T) {
	type C struct {
		I int
		B bool
		S string
		T time.Time
	}
	example := C{
		I: 42,
		B: true,
		S: "Hello",
		T: time.Now(),
	}
	values := map[string][]string{
		"I": {fmt.Sprint(example.I)},
		"B": {fmt.Sprint(example.B)},
		"S": {fmt.Sprint(example.S)},
		"T": {example.T.Format(time.RFC3339)},
	}

	r, warn, err := Decode[C](values)
	if err != nil || warn != nil {
		t.Fatal(warn, err)
	}
	fmt.Println(r)

}
