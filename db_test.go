package notmuch

// Copyright © 2015 The go.notmuch Authors. Authors can be found in the AUTHORS file.
// Licensed under the GPLv3 or later.
// See COPYING at the root of the repository for details.

import (
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"
	"testing"
)

var dbPath string

func init() {
	var err error
	dbPath, err = filepath.Abs("fixtures/database-v1")
	if err != nil {
		panic(err)
	}
}

func TestOpenNotFound(t *testing.T) {
	_, err := Open("/not-found", DBReadOnly)
	if err == nil {
		t.Errorf("Open(%q): expected error got nil", "/not-found")
	}
}

func TestCreate(t *testing.T) {
	tmp := os.TempDir()
	tmpDbPath := path.Join(tmp, ".notmuch")
	defer func() {
		os.RemoveAll(tmpDbPath)
	}()

	db, err := Create(tmp)
	if err != nil {
		t.Fatal(err)
	}
	if want, got := 1, db.Version(); got < want {
		t.Errorf("db.Version(): want at least %d got %d", want, got)
	}
}

func TestOpen(t *testing.T) {
	db, err := Open(dbPath, DBReadOnly)
	if err != nil {
		t.Fatalf("Open(%q): unexpected error: %s", dbPath, err)
	}
	defer db.Close()
	if want, got := 1, db.Version(); got < want {
		t.Errorf("db.Version(): want at least %d got %d", want, got)
	}
}

func TestLastStatus(t *testing.T) {
	db, err := Open(dbPath, DBReadOnly)
	if err != nil {
		t.Fatalf("Open(%q): unexpected error: %s", dbPath, err)
	}
	defer db.Close()
	if want, got := "", db.LastStatus(); want != got {
		t.Errorf("db.LastStatus(): want %s got %s", want, got)
	}
	// TODO(kalbasit): use add_message later to cause an error and add a test for it
}

func TestPath(t *testing.T) {
	db, err := Open(dbPath, DBReadOnly)
	if err != nil {
		t.Fatalf("Open(%q): unexpected error: %s", dbPath, err)
	}
	defer db.Close()
	if want, got := "fixtures/database-v1", db.Path(); !strings.HasSuffix(got, want) {
		t.Errorf("db.Path(): want %s got %s", want, got)
	}
}

func TestNeedsUpgrade(t *testing.T) {
	db, err := Open(dbPath, DBReadOnly)
	if err != nil {
		t.Fatalf("Open(%q): unexpected error: %s", dbPath, err)
	}
	defer db.Close()
	if want, got := false, db.NeedsUpgrade(); want != got {
		t.Errorf("db.NeedsUpgrade(): want %t got %t", want, got)
	}
}

func TestUpgrade(t *testing.T) {
	db, err := Open(dbPath, DBReadOnly)
	if err != nil {
		t.Fatalf("Open(%q): unexpected error: %s", dbPath, err)
	}
	defer db.Close()
	if want, got := ErrReadOnlyDB, db.Upgrade(); want != got {
		t.Errorf("db.Upgrade(): want error %q got %q", want, got)
	}

	db, err = Open(dbPath, DBReadWrite)
	if err != nil {
		t.Fatalf("Open(%q): unexpected error: %s", dbPath, err)
	}
	defer db.Close()
	if want, got := error(nil), db.Upgrade(); want != got {
		t.Errorf("db.Upgrade(): want error %q got %q", want, got)
	}
}

func TestAddMessage(t *testing.T) {
	fp, err := filepath.Abs("fixtures/emails/notmuch0202.2,")
	if err != nil {
		t.Fatalf("error getting the absolute path: %s", err)
	}
	nfp, err := filepath.Abs("fixtures/database-v1/new/notmuch0202.2,")
	if err != nil {
		t.Fatalf("error getting the absolute path: %s", err)
	}

	f, err := os.Open(fp)
	if err != nil {
		t.Fatalf("error opening the new email: %s", err)
	}
	defer f.Close()
	nf, err := os.Create(nfp)
	if err != nil {
		t.Fatalf("error creating the new email: %s", err)
	}
	defer nf.Close()
	if _, err := io.Copy(nf, f); err != nil {
		t.Fatalf("error copying the email: %s", err)
	}
	defer os.Remove(nfp)

	db, err := Open(dbPath, DBReadWrite)
	if err != nil {
		t.Fatalf("Open(%q): unexpected error: %s", dbPath, err)
	}
	defer db.Close()
	msg, err := db.AddMessage(nfp)
	if err != nil {
		t.Fatalf("AddMessage(%q): got error: %s", nfp, err)
	}
	defer db.RemoveMessage(nfp)
	if msg == nil {
		t.Errorf("expecting msg to not be nil")
	}
}

func testFindMessage(t *testing.T, db *DB, id string) {
	msg, err := db.FindMessage(id)
	if err != nil {
		t.Fatalf("db.FindMessage(%q): unexpected error: %s", id, err)
	}
	if want, got := id, msg.ID(); want != got {
		t.Errorf("db.FindMessage(%q).ID(): want %s got %s", id, want, got)
	}
}

func TestFindMessage(t *testing.T) {
	db, err := Open(dbPath, DBReadOnly)
	if err != nil {
		t.Fatalf("Open(%q): unexpected error: %s", dbPath, err)
	}
	defer db.Close()
	if _, err := db.FindMessage("notfound"); err != ErrNotFound {
		t.Errorf("db.FindMessage(\"notfound\"): expecting ErrNotFound got %s", err)
	}
	id := "87iqd9rn3l.fsf@vertex.dottedmag"
	testFindMessage(t, db, id)
}

func TestCompact(t *testing.T) {
	backup := fmt.Sprintf("%s.backup", dbPath)
	defer os.RemoveAll(backup)
	if err := Compact(dbPath, backup); err != nil {
		t.Fatalf("error compacting %q: %s", dbPath, err)
	}

	db, err := Open(dbPath, DBReadOnly)
	if err != nil {
		t.Fatalf("Open(%q): unexpected error: %s", dbPath, err)
	}
	defer db.Close()
	id := "87iqd9rn3l.fsf@vertex.dottedmag"
	testFindMessage(t, db, id)
}

func TestFindMessageByFilename(t *testing.T) {
	db, err := Open(dbPath, DBReadOnly)
	if err != nil {
		t.Fatalf("Open(%q): unexpected error: %s", dbPath, err)
	}
	defer db.Close()
	if _, err := db.FindMessageByFilename("notfound"); err != ErrNotFound {
		t.Errorf("db.FindMessageByFilename(\"notfound\"): expecting ErrNotFound got %s", err)
	}
	id := "87iqd9rn3l.fsf@vertex.dottedmag"
	p := path.Join(dbPath, "new", "04:2,")
	msg, err := db.FindMessageByFilename(p)
	if err != nil {
		t.Fatalf("db.FindMessageByFilename(%q): unexpected error: %s", p, err)
	}
	if want, got := id, msg.ID(); want != got {
		t.Errorf("db.FindMessageByFilename(%q).ID(): want %s got %s", p, want, got)
	}
}

func TestTags(t *testing.T) {
	db, err := Open(dbPath, DBReadOnly)
	if err != nil {
		t.Fatalf("Open(%q): unexpected error: %s", dbPath, err)
	}
	defer db.Close()
	if _, err := db.Tags(); err != nil {
		t.Fatalf("db.Tags(): got error %s", err)
	}
	// TODO(kalbasit): extend the test when tags are fully implemented.
}

func TestGetConfigList(t *testing.T) {
	db, err := Open(dbPath, DBReadOnly)
	if err != nil {
		t.Fatalf("Open(%q): unexpected error: %s", dbPath, err)
	}
	defer db.Close()
	if _, err := db.GetConfigList(""); err != nil {
		t.Errorf("db.GetConfigList(\"\"): unexpected error %s", err)
	}
}

func TestSetConfig(t *testing.T) {
	db, err := Open(dbPath, DBReadWrite)
	if err != nil {
		t.Fatalf("Open(%q): unexpected error: %s", dbPath, err)
	}
	defer db.Close()
	cfgKey := "search.exclude_tags"
	cfgVal := "spam"
	if err := db.SetConfig(cfgKey, cfgVal); err != nil {
		t.Errorf("db.SetConfig(%q, %q): unexpected error %s", cfgKey, cfgVal, err)
	}
}

func TestGetConfig(t *testing.T) {
	db, err := Open(dbPath, DBReadWrite)
	if err != nil {
		t.Fatalf("Open(%q): unexpected error: %s", dbPath, err)
	}
	defer db.Close()
	if _, err := db.GetConfig("blah"); err != nil {
		t.Errorf("db.GetConfig(\"blah\"): unexpected error %s", err)
	}
}

func TestConfigRoundtrip(t *testing.T) {
	db, err := Open(dbPath, DBReadWrite)
	if err != nil {
		t.Fatalf("Open(%q): unexpected error: %s", dbPath, err)
	}
	defer db.Close()
	cfgKey := "search.exclude_tags"
	cfgVal := "spam"
	if err := db.SetConfig(cfgKey, cfgVal); err != nil {
		t.Errorf("db.SetConfig(%q, %q): unexpected error %s", cfgKey, cfgVal, err)
	}
	value, err := db.GetConfig(cfgKey)
	if err != nil {
		t.Errorf("db.GetConfig(%q): unexpected error %s", cfgKey, err)
	}
	if value != cfgVal {
		t.Errorf("db.GetConfig(%q): want: %q, got %q", cfgKey, cfgVal, value)
	}
}

func TestConfigListNext(t *testing.T) {
	db, err := Open(dbPath, DBReadWrite)
	if err != nil {
		t.Fatalf("Open(%q): unexpected error: %s", dbPath, err)
	}
	defer db.Close()
	cfgKey := "search.exclude_tags"
	cfgVal := "spam"
	if err := db.SetConfig(cfgKey, cfgVal); err != nil {
		t.Errorf("db.SetConfig(%q, %q): unexpected error %s", cfgKey, cfgVal, err)
	}
	cfgList, err := db.GetConfigList("")
	if err != nil {
		t.Fatalf("db.GetConfigList(%q): unexpected error: %s", "", err)
	}
	var resKey string
	var resVal string
	for cfgList.Next(&resKey, &resVal) {
		break
	}
	if resKey != cfgKey {
		t.Errorf("config key: expected %q, got %q", cfgKey, resKey)
	}
	if resVal != cfgVal {
		t.Errorf("config value: expected %q, got %q", cfgVal, resVal)
	}
	if cfgList.Next(&resKey, &resVal) {
		t.Errorf("iteration did not stop after the end of the options")
	}
}

func TestOpenWithConfig(t *testing.T) {
	// Temp dir for our configs and profiles
	tmpDir, err := os.MkdirTemp("", "notmuch")
	if err != nil {
		t.Fatalf("MkdirTemp(): unexpected error: %s", err)
	}
	defer os.RemoveAll(tmpDir)

	// "Default" profile
	defaultProfileDir := path.Join(tmpDir, "notmuch", "default")
	if err := os.MkdirAll(defaultProfileDir, 0700); err != nil {
		t.Fatalf("MkdirAll(%s): unexpected error: %s", defaultProfileDir, err)
	}

	// "Test" profile
	testProfile := "testProfile"
	testProfileDir := path.Join(tmpDir, "notmuch", testProfile)
	if err := os.MkdirAll(testProfileDir, 0700); err != nil {
		t.Fatalf("MkdirAll(%s): unexpected error: %s", testProfileDir, err)
	}

	for _, dir := range []string{testProfileDir, defaultProfileDir, tmpDir} {
		config, err := os.Create(path.Join(dir, "config"))
		if err != nil {
			t.Fatalf("CreateTemp(%s): unexpected error: %s", dir, err)
		}
		defer config.Close()
		if _, err := config.WriteString(fmt.Sprintf("[database]\npath=%s\n", dbPath)); err != nil {
			t.Fatalf("WriteString(%q): unexpected error: %s", config.Name(), err)
		}
	}

	configPath := path.Join(tmpDir, "config")
	badPath := "/nowherexyz73"
	badProfile := "nowherexyz13"

	type args struct {
		path     *string
		config   *string
		profile  *string
		xdg_home *string
		mode     DBMode
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "with default profile",
			args: args{
				mode:     DBReadOnly,
				xdg_home: &tmpDir,
			},
		},
		{
			name: "with custom profile",
			args: args{
				mode:     DBReadOnly,
				profile:  &testProfile,
				xdg_home: &tmpDir,
			},
		},
		{
			name: "with config file",
			args: args{
				path:   nil,
				config: &configPath,
				mode:   DBReadOnly,
			},
		},
		{
			name: "with db path only",
			args: args{
				path: &dbPath,
				mode: DBReadOnly,
			},
		},
		{
			name: "with non-existent config file",
			args: args{
				config: &badPath,
			},
			wantErr: true,
		},
		{
			name: "with non-existent path",
			args: args{
				path: &badPath,
			},
			wantErr: true,
		},
		{
			name: "with non-existent profile",
			args: args{
				profile:  &badProfile,
				xdg_home: &tmpDir,
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.args.xdg_home != nil {
				os.Setenv("XDG_CONFIG_HOME", *tt.args.xdg_home)
				defer os.Unsetenv("XDG_CONFIG_HOME")
			}

			db, err := OpenWithConfig(tt.args.path, tt.args.config, tt.args.profile, tt.args.mode)
			if (err != nil) != tt.wantErr {
				t.Errorf("OpenWithConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if want, got := 1, db.Version(); got < want {
					t.Errorf("db.Version(): want at least %d got %d", want, got)
				}
			}
		})
	}
}
