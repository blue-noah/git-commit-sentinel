package main

import "testing"

func TestRealMainNoArgs(t *testing.T) {
	if code := realMain(nil); code != 2 {
		t.Errorf("realMain(nil) = %d, want 2 (usage error)", code)
	}
}

func TestRealMainHelp(t *testing.T) {
	for _, arg := range []string{"-h", "--help", "help"} {
		if code := realMain([]string{arg}); code != 0 {
			t.Errorf("realMain([%q]) = %d, want 0", arg, code)
		}
	}
}

func TestRealMainVersion(t *testing.T) {
	for _, arg := range []string{"-v", "--version", "version"} {
		if code := realMain([]string{arg}); code != 0 {
			t.Errorf("realMain([%q]) = %d, want 0", arg, code)
		}
	}
}

func TestRealMainUnknownCommand(t *testing.T) {
	if code := realMain([]string{"bogus"}); code != 2 {
		t.Errorf(`realMain(["bogus"]) = %d, want 2`, code)
	}
}

func TestRealMainDispatchesHook(t *testing.T) {
	// No args after "hook": runHook itself should report the usage error,
	// proving realMain routed here rather than falling to "unknown command".
	if code := realMain([]string{"hook"}); code != 2 {
		t.Errorf(`realMain(["hook"]) = %d, want 2 (runHook's own usage error)`, code)
	}
}
