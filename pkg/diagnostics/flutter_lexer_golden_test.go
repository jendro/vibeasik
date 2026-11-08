package diagnostics

import (
	"strings"
	"testing"
)

const goldenDart = `import 'package:flutter/material.dart';
import 'package:shared_preferences/shared_preferences.dart';

class _HomeState extends State<Home> {
  final _name = TextEditingController();
  late final AnimationController _anim;
  StreamSubscription<int>? _sub;
  Timer? _timer;
  var _node = FocusNode();
  SharedPreferences? _prefs;
  final owned = ScrollController();
  TextEditingController _fromParent;

  @override
  void initState() {
    super.initState();
    _anim = AnimationController(vsync: this);
    _sub = stream.listen((_) {});
    _timer = Timer.periodic(const Duration(seconds: 1), (_) {});
  }

  @override
  void dispose() {
    _name.dispose();
    super.dispose();
  }

  Future<void> _unsafe() async {
    await repo.save();
    Navigator.of(context).pop();
    setState(() {});
    if (!mounted) return;
    setState(() {});
    await foo(
      context,
      1,
    );
    context.go('/x');
    final f = () async {
      await x();
      setState(() {});
    };
  }

  Future<void> _arrow() async => await repo.save();

  @override
  Widget build(BuildContext context) {
    setState(() => _ready = true);
    final data = jsonDecode(raw);
    final ctrl = TextEditingController();
    if (cond) {
      setState(() {});
    }
    print('hi "quoted" ' + "x");
    prefs.setString('auth_token', t);
    return Column(children: [
      ElevatedButton(onPressed: () => setState(() => _n++), child: Text('+')),
      ElevatedButton(onPressed: () {
        setState(() { _n--; });
        final c = TextEditingController();
      }, child: Text('-')),
      FutureBuilder(future: repo.fetchItems(), builder: (ctx, snap) => Text('x')),
      FutureBuilder(future: _items, future2: Future.value(1), builder: (ctx, snap) => Text('y')),
      StreamBuilder(stream: api.watch(), builder: (c, s) { return Text('z'); }),
    ]);
  }

  Widget get title => Text('a');
  Widget build2(BuildContext context) => Text(jsonDecode(x));
  void decl();
  int f() => 1;
}

mixin M {
  final c = StreamController<int>();
  /* block
  print('ignored'); */
  String multi = '''
  print('x');
  ''';
}
`

func TestFlutterLexerGolden(t *testing.T) {
	lines := strings.Split(goldenDart, "\n")
	sl := stripCode(lines, "dart")
	out := map[string]any{}
	out["best_practices"] = inspectFlutterBestPractices(goldenDart, "lib/home.dart", lines)
	out["best_practices_test_path"] = inspectFlutterBestPractices(goldenDart, "test/home_test.dart", lines)
	out["leaks"] = inspectFlutterResourceLeaks(sl, "lib/home.dart", lines)
	noLeak := strings.Split("class A {\n  final x = 1;\n}\n", "\n")
	out["leaks_none"] = inspectFlutterResourceLeaks(stripCode(noLeak, "dart"), "lib/a.dart", noLeak)

	ends := map[string]int{}
	var masks [][][]bool
	for i := range sl {
		ends[intKey(i)] = dartBlockEnd(sl, i)
	}
	out["block_ends"] = ends
	for i := range sl {
		if dartBuildDeclRegex.MatchString(sl[i]) || dartAsyncBodyRegex.MatchString(sl[i]) {
			masks = append(masks, dartClosureMask(sl, i, dartBlockEnd(sl, i)))
		}
	}
	out["masks"] = masks

	// lexer lintas bahasa
	src := strings.Split(strings.Join([]string{
		`a := "str // not" // comment`,
		"b := `raw",
		`multi // x` + "`" + ` + 1 /* c */`,
		`/* block`,
		`still */ code 'q\'s' "d\"e"`,
		`x = """py`,
		`doc""" # hash`,
		`#[Attr] $x = 'u';`,
		`unclosed "string`,
		`'''dart`,
		`end''' tail`,
	}, "\n"), "\n")
	lex := map[string][]string{}
	for _, lang := range []string{"go", "typescript", "python", "php", "dart", "kotlin", "java", "xml", ""} {
		lex[lang] = stripCode(src, lang)
	}
	out["lexer"] = lex
	assertGolden(t, "flutter_lexer", out)
}

func intKey(i int) string {
	return string(rune('0'+i/100)) + string(rune('0'+(i/10)%10)) + string(rune('0'+i%10))
}
