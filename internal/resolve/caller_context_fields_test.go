package resolve

import (
	"reflect"
	"testing"

	"github.com/cfmleditor/clif/internal/vfs"
)

// A caller view is built field by field, so a setting added to Resolver and
// left out of forCaller is silently zero in every view: SnapshotScan was, and
// a cfwheels scan ran at half speed with nothing failing. Every exported field
// set on the resolver must reach its views.
func TestCallerViewsCarryEverySetting(t *testing.T) {
	r := &Resolver{}
	rv := reflect.ValueOf(r).Elem()
	rt := rv.Type()

	for i := range rt.NumField() {
		f := rt.Field(i)
		if !f.IsExported() {
			continue
		}

		field := rv.Field(i)

		switch field.Kind() {
		case reflect.Bool:
			field.SetBool(true)
		case reflect.String:
			field.SetString("x")
		case reflect.Slice:
			field.Set(reflect.MakeSlice(f.Type, 1, 1))
		case reflect.Map:
			field.Set(reflect.MakeMap(f.Type))
		case reflect.Func:
			field.Set(reflect.MakeFunc(f.Type, func([]reflect.Value) []reflect.Value {
				out := make([]reflect.Value, f.Type.NumOut())
				for j := range out {
					out[j] = reflect.Zero(f.Type.Out(j))
				}

				return out
			}))
		case reflect.Pointer:
			field.Set(reflect.New(f.Type.Elem()))
		case reflect.Interface:
			if f.Type != reflect.TypeFor[vfs.FS]() {
				t.Fatalf("field %s: interface %s has no test value; add one", f.Name, f.Type)
			}

			field.Set(reflect.ValueOf(vfs.FS(vfs.OS{})))
		case reflect.Struct:
			// A struct of settings: set its first settable int field.
			for _, sub := range field.Fields() {
				if sub.CanSet() && sub.Kind() == reflect.Int {
					sub.SetInt(1)

					break
				}
			}
		default:
			t.Fatalf("field %s: kind %s has no test value; add one", f.Name, field.Kind())
		}
	}

	view := r.forCaller(t.TempDir())
	if view == r {
		t.Fatal("forCaller made no view")
	}

	vv := reflect.ValueOf(view).Elem()

	for i := range rt.NumField() {
		f := rt.Field(i)
		if !f.IsExported() {
			continue
		}

		want, got := rv.Field(i), vv.Field(i)

		var same bool
		if f.Type.Kind() == reflect.Func || f.Type.Kind() == reflect.Map || f.Type.Kind() == reflect.Pointer || f.Type.Kind() == reflect.Slice {
			same = want.Pointer() == got.Pointer()
		} else {
			same = reflect.DeepEqual(want.Interface(), got.Interface())
		}

		if !same {
			t.Errorf("forCaller does not carry %s into the view", f.Name)
		}
	}
}
