package diagnostics

import (
	"strings"
	"testing"
)

const flutterPubspec = "name: app\ndependencies:\n  flutter:\n    sdk: flutter\n"

func TestFlutterBuildRules(t *testing.T) {
	dart := `import 'package:flutter/material.dart';

class _HomeState extends State<Home> {
  @override
  Widget build(BuildContext context) {
    setState(() => _ready = true);
    final data = jsonDecode(raw);
    final ctrl = TextEditingController();
    return Column(children: [
      ElevatedButton(onPressed: () => setState(() => _n++), child: Text('+')),
      ElevatedButton(onPressed: () {
        setState(() { _n--; });
        final c = TextEditingController();
      }, child: Text('-')),
      FutureBuilder(future: repo.fetchItems(), builder: (ctx, snap) => Text('x')),
      FutureBuilder(future: _items, builder: (ctx, snap) => Text('y')),
    ]);
  }
}
`
	got := ruleSet(InspectFile(writeFixture(t, t.TempDir(), "lib/home.dart", dart), "lib/home.dart", "dart"))
	for id, line := range map[string]int{"MOB-SETSTATE-IN-BUILD": 6, "MOB-UI-THREAD-FREEZE": 7, "MOB-CONTROLLER-IN-BUILD": 8, "MOB-FUTURE-IN-BUILD": 15} {
		f, ok := got[id]
		if !ok {
			t.Errorf("expected %s", id)
			continue
		}
		if f.Line != line {
			t.Errorf("%s: expected line %d, got %d", id, line, f.Line)
		}
		if strings.Contains(f.Description, " 2 kali") {
			t.Errorf("%s: closure usages (onPressed) must not be counted: %s", id, f.Description)
		}
	}
}

func TestFlutterAsyncGap(t *testing.T) {
	dart := `import 'package:flutter/material.dart';

class _S extends State<S> {
  Future<void> _safe() async {
    final ok = await Navigator.push<bool>(
      context,
      MaterialPageRoute(builder: (_) => const Edit()),
    );
    if (!mounted) return;
    setState(() => _ok = ok);
  }

  Future<void> _unsafe() async {
    await repo.save();
    Navigator.of(context).pop();
  }
}
`
	f, ok := ruleSet(InspectFile(writeFixture(t, t.TempDir(), "lib/s.dart", dart), "lib/s.dart", "dart"))["MOB-ASYNC-GAP-CONTEXT"]
	if !ok || f.Line != 15 || !strings.Contains(f.Description, " 1 kali") {
		t.Errorf("expected exactly the unsafe context use on line 15, got %+v", f)
	}
}

func TestFlutterResourceLeak(t *testing.T) {
	dart := `import 'package:flutter/material.dart';

class Field extends StatelessWidget {
  final TextEditingController controller;
  const Field({super.key, required this.controller});
}

class _FormState extends State<FormPage> {
  final _name = TextEditingController();
  final _email = TextEditingController();
  late final AnimationController _anim;
  StreamSubscription<int>? _sub;
  Timer? _timer;

  @override
  void initState() {
    super.initState();
    _anim = AnimationController(vsync: this);
    _sub = stream.listen((_) {});
    _timer = Timer.periodic(const Duration(seconds: 1), (_) {});
  }

  @override
  void dispose() {
    for (final c in [_name, _email]) {
      c.dispose();
    }
    _anim.dispose();
    super.dispose();
  }
}
`
	f, ok := ruleSet(InspectFile(writeFixture(t, t.TempDir(), "lib/form.dart", dart), "lib/form.dart", "dart"))["MOB-CONTROLLER-LEAK"]
	if !ok {
		t.Fatalf("expected MOB-CONTROLLER-LEAK")
	}
	if f.Title != "Flutter Resource Tidak Dilepas: _sub, _timer" {
		t.Errorf("only _sub and _timer leak (injected controller & list-disposed fields are fine), got %q", f.Title)
	}
}

func TestFlutterInsecureStorageAndPrint(t *testing.T) {
	dart := `import 'package:shared_preferences/shared_preferences.dart';

Future<void> save(SharedPreferences prefs, String t) async {
  await prefs.setString('access_token', t);
  await prefs.setString(AppKeys.authToken, t);
  await prefs.setString('shipping_address', t);
  print('saved');
  debugPrint('ok');
}
`
	dir := t.TempDir()
	writeFixture(t, dir, "pubspec.yaml", flutterPubspec)
	got := ruleSet(InspectFile(writeFixture(t, dir, "lib/store.dart", dart), "lib/store.dart", "dart"))
	if f, ok := got["MOB-INSECURE-LOCAL-STORAGE"]; !ok || f.Line != 4 || !strings.Contains(f.Description, " 2 kali") {
		t.Errorf("expected 2 sensitive setString (not shipping_address), got %+v", f)
	}
	if f, ok := got["MOB-PRINT-STATEMENT"]; !ok || f.Line != 7 {
		t.Errorf("expected print() on line 7, got %+v", f)
	}
}

func TestFlutterRulesSkipPureDart(t *testing.T) {
	dart := `import 'package:shelf/shelf.dart';

class Api {
  final _ctrl = StreamController<int>();
  Future<void> handle() async {
    await db.save();
    print('saved');
  }
}
`
	dir := t.TempDir()
	writeFixture(t, dir, "pubspec.yaml", "name: api\ndependencies:\n  shelf: ^1.4.0\n")
	for id := range ruleSet(InspectFile(writeFixture(t, dir, "lib/screens/api_screen.dart", dart), "lib/screens/api_screen.dart", "dart")) {
		if strings.HasPrefix(id, "MOB-") {
			t.Errorf("Flutter rule %s must not run on pure Dart", id)
		}
	}
}

func TestFlutterProjectLints(t *testing.T) {
	dir := t.TempDir()
	writeFixture(t, dir, "pubspec.yaml", flutterPubspec)
	writeFixture(t, dir, "lib/main.dart", "import 'package:flutter/material.dart';\nvoid main() => runApp(const App());\n")
	if _, ok := ruleSet(InspectDirectory(dir, "mobile"))["MOB-NO-FLUTTER-LINTS"]; !ok {
		t.Errorf("expected MOB-NO-FLUTTER-LINTS without analysis_options.yaml")
	}
	writeFixture(t, dir, "analysis_options.yaml", "include: package:flutter_lints/flutter.yaml\n")
	if _, ok := ruleSet(InspectDirectory(dir, "mobile"))["MOB-NO-FLUTTER-LINTS"]; ok {
		t.Errorf("flutter_lints include must satisfy the rule")
	}
}
