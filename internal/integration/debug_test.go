//go:build integration

package integration

import (
	"flag"
	"testing"
)

func TestFlagParsing(t *testing.T) {
	// Flags must appear before positional arguments for flag package to recognize them.
	args := []string{"--test", "echo 'test passed'", "test description"}
	fs := flag.NewFlagSet("task-new", flag.ExitOnError)
	testFlag := fs.String("test", "", "Test command")
	// parent and help flags omitted for brevity

	if err := fs.Parse(args); err != nil {
		t.Fatal(err)
	}

	if *testFlag != "echo 'test passed'" {
		t.Errorf("testFlag mismatch: got %q", *testFlag)
	}
	if fs.Arg(0) != "test description" {
		t.Errorf("description mismatch: got %q", fs.Arg(0))
	}
}
