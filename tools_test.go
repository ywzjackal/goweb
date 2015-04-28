package goweb

import (
	"net/url"
	"reflect"
	"testing"
)

type TestCCM struct {
	Str  string
	Strs []string
	I    int
	Is   []int
	B    bool
	Bs   []bool
	F    float32
	Fs   []float32
}

func Test_ParamtersFromRequestUrl(t *testing.T) {
	urlString := "http://aa.bb.com:8080/to/path?Str=str;Strs=1;Strs=2;I=102;Is=1;Is=2;B=True;Bs=True;Bs=False;F=1.03;Fs=1.1;Fs=1.2"
	u, err := url.Parse(urlString)
	if err != nil {
		t.Error(err)
		return
	}
	values := u.Query()
	result := paramtersFromRequestUrl(reflect.TypeOf(&TestCCM{}), values).Interface()
	if rt, ok := result.(*TestCCM); ok {
		t.Logf("RT:% +v", rt)
		if rt.B != true || len(rt.Bs) != 2 || rt.Bs[0] != true || rt.Bs[1] != false {
			t.Error("Fail to parse bool(s) value!")
			return
		}
		if rt.Str != "str" || len(rt.Strs) != 2 || rt.Strs[0] != "1" || rt.Strs[1] != "2" {
			t.Error("Fail to parse string(s) value!")
			return
		}
		if rt.F != 1.03 || len(rt.Fs) != 2 || rt.Fs[0] != 1.1 || rt.Fs[1] != 1.2 {
			t.Error("Fail to parse flost(s) value!")
			return
		}
		if rt.I != 102 || len(rt.Is) != 2 || rt.Is[0] != 1 || rt.Is[1] != 2 {
			t.Error("Fail to parse int(s) value!")
			return
		}
		return
	}
	t.Fail()
}
