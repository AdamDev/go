// Copyright 2021 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package godebug_test

import (
	"fmt"
	. "internal/godebug"
	"internal/testenv"
	"os"
	"os/exec"
	"reflect"
	"runtime/metrics"
	"sort"
	"strings"
	"testing"
)

func TestGet(t *testing.T) {
	foo := New("#foo")
	tests := []struct {
		godebug string
		setting *Setting
		want    string
	}{
		{"", New("#"), ""},
		{"", foo, ""},
		{"foo=bar", foo, "bar"},
		{"foo=bar,after=x", foo, "bar"},
		{"before=x,foo=bar,after=x", foo, "bar"},
		{"before=x,foo=bar", foo, "bar"},
		{",,,foo=bar,,,", foo, "bar"},
		{"foodecoy=wrong,foo=bar", foo, "bar"},
		{"foo=", foo, ""},
		{"foo", foo, ""},
		{",foo", foo, ""},
		{"foo=bar,baz", New("#loooooooong"), ""},
	}
	for _, tt := range tests {
		t.Setenv("GODEBUG", tt.godebug)
		got := tt.setting.Value()
		if got != tt.want {
			t.Errorf("get(%q, %q) = %q; want %q", tt.godebug, tt.setting.Name(), got, tt.want)
		}
	}
}

func TestMetrics(t *testing.T) {
	const name = "http2client" // must be a real name so runtime will accept it

	var m [1]metrics.Sample
	m[0].Name = "/godebug/non-default-behavior/" + name + ":events"
	metrics.Read(m[:])
	if kind := m[0].Value.Kind(); kind != metrics.KindUint64 {
		t.Fatalf("NonDefault kind = %v, want uint64", kind)
	}

	s := New(name)
	s.Value()
	s.IncNonDefault()
	s.IncNonDefault()
	s.IncNonDefault()
	metrics.Read(m[:])
	if kind := m[0].Value.Kind(); kind != metrics.KindUint64 {
		t.Fatalf("NonDefault kind = %v, want uint64", kind)
	}
	if count := m[0].Value.Uint64(); count != 3 {
		t.Fatalf("NonDefault value = %d, want 3", count)
	}
}

func TestCmdBisect(t *testing.T) {
	testenv.MustHaveGoBuild(t)
	out, err := exec.Command("go", "run", "cmd/vendor/golang.org/x/tools/cmd/bisect", "GODEBUG=buggy=1#PATTERN", os.Args[0], "-test.run=BisectTestCase").CombinedOutput()
	if err != nil {
		t.Fatalf("exec bisect: %v\n%s", err, out)
	}

	var want []string
	src, err := os.ReadFile("godebug_test.go")
	for i, line := range strings.Split(string(src), "\n") {
		if strings.Contains(line, "BISECT"+" "+"BUG") {
			want = append(want, fmt.Sprintf("godebug_test.go:%d", i+1))
		}
	}
	sort.Strings(want)

	var have []string
	for _, line := range strings.Split(string(out), "\n") {
		if strings.Contains(line, "godebug_test.go:") {
			have = append(have, line[strings.LastIndex(line, "godebug_test.go:"):])
		}
	}
	sort.Strings(have)

	if !reflect.DeepEqual(have, want) {
		t.Errorf("bad bisect output:\nhave %v\nwant %v\ncomplete output:\n%s", have, want, string(out))
	}
}

// This test does nothing by itself, but you can run
//	bisect 'GODEBUG=buggy=1#PATTERN' go test -run=BisectTestCase
// to see that the GODEBUG bisect support is working.
// TestCmdBisect above does exactly that.
func TestBisectTestCase(t *testing.T) {
	s := New("#buggy")
	for i := 0; i < 10; i++ {
		if s.Value() == "1" {
			t.Log("ok")
		}
		if s.Value() == "1" {
			t.Log("ok")
		}
		if s.Value() == "1" { // BISECT BUG
			t.Error("bug")
		}
		if s.Value() == "1" && // BISECT BUG
			s.Value() == "1" { // BISECT BUG
			t.Error("bug")
		}
	}
}
