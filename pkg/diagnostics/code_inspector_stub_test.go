package diagnostics

import (
	"strings"
	"testing"
)

func TestStubHandlerRegexPrecision(t *testing.T) {
	cases := []struct {
		line string
		want bool
	}{
		{`// TODO: implement this`, true},
		{`panic("not implemented")`, true},
		{`return c.Status(501).SendString("status: not implemented")`, true},
		{`return c.SendStatus(http.StatusNotImplemented)`, true},
		{`throw new NotImplementedError();`, true},
		{`throw NotImplementedException(`, true},
		{`throwMyNotImplementedError`, false},
		{`throw new NotImplementedErrorX`, false},
		{`// handle the request`, false},
		{`handleFoo()`, false},
		{`return ` + strings.Repeat("x", 100) + ` not implemented`, false},
	}
	for _, c := range cases {
		if got := stubHandlerRegex.MatchString(c.line); got != c.want {
			t.Errorf("%q: got %v want %v", c.line, got, c.want)
		}
	}
}
