package diagnostics

import "testing"

func TestGeneratedFlutterFilesSkipped(t *testing.T) {
	if !isTestOrGeneratedFile("GeneratedPluginRegistrant.java") {
		t.Error("GeneratedPluginRegistrant.java harus dianggap hasil generator")
	}
	if !isGeneratedContent("package io.flutter.plugins;\n/**\n * Generated file. Do not edit.\n */") {
		t.Error("header 'Do not edit' (huruf campuran) harus dikenali")
	}
	if !ignoredDirNames["ephemeral"] {
		t.Error("folder ephemeral Flutter harus dilewati")
	}
	if isGeneratedContent("package main\n\nfunc main() {}\n") {
		t.Error("kode biasa bukan hasil generator")
	}
}
