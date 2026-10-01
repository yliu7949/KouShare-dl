package test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"
	cmdks "github.com/yliu7949/KouShare-dl/cmd/ks"
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
		"save":   cmdks.SaveCmd,
		"record": cmdks.RecordCmd,
		"merge":  cmdks.MergeCmd,
		"slide":  cmdks.SlideCmd,
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

func TestEmptyPathUsesCurrentDirectory(t *testing.T) {
	directory := t.TempDir()
	originalDirectory, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(directory); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(originalDirectory); err != nil {
			t.Errorf("restore working directory: %v", err)
		}
	})
	temporaryFile := filepath.Join(directory, "download.tmp")
	if err := os.WriteFile(temporaryFile, []byte("temporary"), 0600); err != nil {
		t.Fatal(err)
	}

	command := cmdks.CleanCmd()
	command.SetArgs([]string{"--path", "", "--quiet"})
	if err := command.Execute(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(temporaryFile); !os.IsNotExist(err) {
		t.Fatalf("empty --path did not clean the current directory: %v", err)
	}
}

func TestLegacyPositionalArgumentsRemainAccepted(t *testing.T) {
	tests := []struct {
		name string
		args []string
		new  commandFactory
	}{
		{name: "info", args: []string{"7304"}, new: cmdks.InfoCmd},
		{name: "save", args: []string{"7304"}, new: cmdks.SaveCmd},
		{name: "record", args: []string{"751111"}, new: cmdks.RecordCmd},
		{name: "merge", args: []string{"."}, new: cmdks.MergeCmd},
		{name: "slide", args: []string{"7405"}, new: cmdks.SlideCmd},
		{name: "login with phone", args: []string{"13800138000"}, new: cmdks.LoginCmd},
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
