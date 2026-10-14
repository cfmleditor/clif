package resolve

import (
	"reflect"
	"testing"

	"github.com/cfmleditor/clif/internal/index"
	"github.com/cfmleditor/clif/internal/vfs"
)

// fill sets v, and anything inside it, to something non-zero, so that a field a
// copy forgets reads as different from the original.
func fill(t *testing.T, v reflect.Value) {
	t.Helper()

	switch v.Kind() {
	case reflect.String:
		v.SetString("x")
	case reflect.Bool:
		v.SetBool(true)
	case reflect.Int, reflect.Int64:
		v.SetInt(7)
	case reflect.Map:
		m := reflect.MakeMap(v.Type())
		k, e := reflect.New(v.Type().Key()).Elem(), reflect.New(v.Type().Elem()).Elem()

		fill(t, k)
		fill(t, e)
		m.SetMapIndex(k, e)
		v.Set(m)
	case reflect.Slice:
		s := reflect.MakeSlice(v.Type(), 1, 1)
		fill(t, s.Index(0))
		v.Set(s)
	case reflect.Func:
		v.Set(reflect.MakeFunc(v.Type(), func([]reflect.Value) []reflect.Value {
			out := make([]reflect.Value, v.Type().NumOut())
			for i := range out {
				out[i] = reflect.Zero(v.Type().Out(i))
			}

			return out
		}))
	case reflect.Pointer:
		p := reflect.New(v.Type().Elem())
		if p.Elem().Kind() == reflect.Struct {
			fill(t, p.Elem())
		}

		v.Set(p)
	case reflect.Struct:
		for i := range v.NumField() {
			if v.Type().Field(i).IsExported() {
				fill(t, v.Field(i))
			}
		}
	case reflect.Interface:
		if v.Type() == reflect.TypeFor[vfs.FS]() {
			v.Set(reflect.ValueOf(vfs.OS{}))

			return
		}

		t.Fatalf("no value to fill %s with", v.Type())
	default:
		t.Fatalf("no value to fill kind %s", v.Kind())
	}
}

// TestACallerContextViewCarriesEveryField: a view of the resolver for another
// application (forCaller) is a copy written out field by field, and a field
// added to Resolver and not to the copy is a setting the view silently lacks.
// That happened when the parse configuration was added, and the only symptom
// was a resolution that differed between two applications' calls. Every
// exported field is set to something and must come out the other side.
func TestACallerContextViewCarriesEveryField(t *testing.T) {
	r := &Resolver{}
	rv := reflect.ValueOf(r).Elem()

	for i := range rv.NumField() {
		if rv.Type().Field(i).IsExported() {
			fill(t, rv.Field(i))
		}
	}

	r.Index = index.New()
	r.FS = vfs.OS{}
	r.WorkspaceFolders = []string{t.TempDir()}

	view := r.forCaller(r.WorkspaceFolders[0])
	if view == r {
		t.Skip("the resolver served itself, so there is no copy to check")
	}

	vv := reflect.ValueOf(view).Elem()

	for i := range rv.NumField() {
		f := rv.Type().Field(i)
		if !f.IsExported() {
			continue
		}

		a, b := rv.Field(i), vv.Field(i)

		if a.Kind() == reflect.Func {
			if a.Pointer() != b.Pointer() {
				t.Errorf("the caller-context view does not carry %s", f.Name)
			}

			continue
		}

		if !reflect.DeepEqual(a.Interface(), b.Interface()) {
			t.Errorf("the caller-context view does not carry %s: the original has %v, the view %v", f.Name, a.Interface(), b.Interface())
		}
	}
}
