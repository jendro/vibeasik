package diagnostics

import (
	"strings"
	"testing"
)

type cxGolden struct {
	Name  string
	Line  int
	Count int
}

// Test karakterisasi measureFunctions untuk setiap profil bahasa.
func TestMeasureFunctionsGolden(t *testing.T) {
	src := map[string]string{
		"go": `package x
func a(x int) int {
	if x > 1 && x < 5 || x == 9 {
		for i := 0; i < x; i++ {
			switch i {
			case 1:
			case 2:
			}
		}
	}
	f := func() {
		if x > 0 {
		}
	}
	return 1
}

func (r *R) b() {}
func c[T any](x T) {}
type I interface {
	M()
}
var s = "if for case"
// if x
func d() { if a { } }
`,
		"typescript": `export function a(x: number): number {
  if (x > 1 && x < 5) { return 1 }
  const f = (y) => y ? 1 : 2
  return x ?? 3
}
export const g = async (a, b) =>
  a && b
const h = (z: number): number => {
  for (const q of z) { if (q) {} }
  return 1
}
class K {
  public async m(x) {
    while (x) { x-- }
  }
  abstract n(): void;
  get v() { return 1 }
}
const s = "if (x) {" // for
`,
		"kotlin": `class A {
    fun a(x: Int): Int {
        if (x > 1 && x < 5) return 1
        return when (x) {
            1 -> 2
            else -> 3
        }
    }
    private suspend fun <T> b(x: T) = x ?: 0
    override fun c(): Int = if (z) 1 else 2
    fun d()
}
`,
		"java": `public class A {
    public static int a(int x) {
        if (x > 1 && x < 5) { return 1; }
        for (int i = 0; i < x; i++) { x += i > 2 ? 1 : 0; }
        try { } catch (Exception e) { }
        return x;
    }
    public abstract void b();
    private int c(int y) { switch (y) { case 1: return 1; default: return 2; } }
}
`,
		"dart": `class A {
  int a(int x) {
    if (x > 1 && x < 5) { return 1; }
    return x > 3 ? 1 : 2;
  }
  String get name => cond ? 'a' : 'b';
  void b() {
    return foo(1);
    await bar(2);
    final x = baz(3);
  }
  Future<void> c() async {
    for (final i in list) { if (i) {} }
  }
  static int d(int v) => v > 1 && v < 3 ? 1 : 0;
}
`,
		"php": `<?php
class A {
    public function a($x) {
        if ($x > 1 && $x < 5) { return 1; }
        foreach ($l as $i) { if ($i) {} elseif ($j) {} }
        return $x ? 1 : 2;
    }
    private static function &b() { try { } catch (E $e) { } }
}
function c() { while (1) {} }
`,
		"python": `def a(x):
    if x > 1 and x < 5 or x == 0:
        for i in x:
            pass
    elif x:
        pass
    return 1

class K:
    async def m(self, y):
        try:
            pass
        except Exception:
            pass
        def inner(z):
            if z:
                return 1
        return inner(
            y
        )

def b(): pass
`,
	}
	out := map[string][]cxGolden{}
	for lang, body := range src {
		profile := complexityProfiles[lang]
		var res []cxGolden
		for _, f := range measureFunctions(strings.Split(body, "\n"), lang, profile) {
			res = append(res, cxGolden{f.name, f.line, f.count})
		}
		out[lang] = res
	}
	assertGolden(t, "complexity", out)

	// matchFuncStart langsung
	ms := map[string]string{}
	for _, c := range []struct{ lang, line string }{
		{"go", "func (r *R) Foo[T any](x T) {"}, {"go", "func main() {"}, {"go", "x := func() {"},
		{"typescript", "export default async function* gen<T>(x) {"}, {"typescript", "  if (a) {"}, {"typescript", "  while (x) {"},
		{"typescript", "  async foo(a: number): Promise<void> {"}, {"typescript", "const f = async (a) =>"},
		{"dart", "  Future<void> load(int a) {"}, {"dart", "  return foo(1);"}, {"dart", "  final x = foo(1);"}, {"dart", "  void a();"}, {"dart", "  await bar(2) {"},
		{"python", "  async def go(self):"}, {"kotlin", "  override fun <T> a.b(x: T): Int {"}, {"java", "  public static void main(String[] a) {"},
	} {
		ms[c.lang+"|"+c.line] = matchFuncStart(complexityProfiles[c.lang], c.lang, c.line)
	}
	assertGolden(t, "match_func_start", ms)
}
