package ks

import (
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"
)

func TestLegacyCommandAliasesAndFlags(t *testing.T) {
	tests := []struct {
		name    string
		aliases []string
		flags   []string
	}{
		{name: "save", aliases: []string{"video"}, flags: []string{"path", "quality", "series", "vidPrefix", "concurrency", "video-concurrency"}},
		{name: "record", aliases: []string{"live"}, flags: []string{"path", "at", "autoMerge", "replay", "password"}},
		{name: "merge", flags: []string{"name"}},
		{name: "slide", flags: []string{"path", "series", "qpdf-bin"}},
	}

	commands := map[string]commandFactory{
		"save":   SaveCmd,
		"record": RecordCmd,
		"merge":  MergeCmd,
		"slide":  SlideCmd,
	}
	for _, test := range tests {
		command := commands[test.name]()
		for _, alias := range test.aliases {
			if !contains(command.Aliases, alias) {
				t.Errorf("%s aliases %v do not contain legacy alias %q", test.name, command.Aliases, alias)
			}
		}
		for _, flag := range test.flags {
			if command.Flags().Lookup(flag) == nil && command.PersistentFlags().Lookup(flag) == nil {
				t.Errorf("%s no longer exposes legacy flag --%s", test.name, flag)
			}
		}
	}
}

func TestNormalizedDirectory(t *testing.T) {
	if got := normalizedDirectory(""); got != "." {
		t.Fatalf("normalizedDirectory(empty) = %q", got)
	}
	want := filepath.Join("parent", "child")
	if got := normalizedDirectory("parent/child/"); got != want {
		t.Fatalf("normalizedDirectory = %q, want %q", got, want)
	}
}

func TestLegacyPositionalArgumentsRemainAccepted(t *testing.T) {
	tests := []struct {
		name string
		args []string
		new  commandFactory
	}{
		{name: "info", args: []string{"7304"}, new: InfoCmd},
		{name: "save", args: []string{"7304"}, new: SaveCmd},
		{name: "record", args: []string{"751111"}, new: RecordCmd},
		{name: "merge", args: []string{"."}, new: MergeCmd},
		{name: "slide", args: []string{"7405"}, new: SlideCmd},
		{name: "login with phone", args: []string{"13800138000"}, new: LoginCmd},
	}
	for _, test := range tests {
		command := test.new()
		if err := command.Args(command, test.args); err != nil {
			t.Errorf("legacy %s arguments %v rejected: %v", test.name, test.args, err)
		}
	}
}

type commandFactory func() *cobra.Command

func contains(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}
