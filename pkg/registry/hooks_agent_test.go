package registry

import (
	"errors"
	"testing"
)

func TestWrapAgentCommand(t *testing.T) {
	old := H
	t.Cleanup(func() { H = old })
	H.AgentCommand = nil
	p, a, e, err := WrapAgentCommand("/bin/x", []string{"a"}, "/d", []string{"K=V"})
	if err != nil || p != "/bin/x" || len(a) != 1 || a[0] != "a" || len(e) != 1 {
		t.Fatalf("tanpa hook masukan harus utuh: %v %v %v %v", p, a, e, err)
	}
	H.AgentCommand = func(string, []string, string, []string) (string, []string, []string, error) {
		return "", nil, nil, errors.New("tolak")
	}
	if _, _, _, err := WrapAgentCommand("/bin/x", nil, "/d", nil); err == nil {
		t.Fatal("error hook harus diteruskan")
	}
}
