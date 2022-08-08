package procfs_test

import (
	"bufio"
	. "github.com/instana/envcheck/procfs"
	"os"
	"path"
	"strings"
	"testing"
)

func Test_should_walk_process_stats(t *testing.T) {
	root := generateProcFS(t)
	stats, err := ReadStats(root)
	if err != nil {
		t.Fatalf("err=%v", err)
	}

	if len(stats) != 107 {
		t.Errorf("got len=%v, want 107", len(stats))
	}

	if stats[0].Name != "(systemd)" {
		t.Errorf("got `%v`, want ``", stats[0].Name)
	}
}

func Test_should_process_line(t *testing.T) {
	r := strings.NewReader(`97 (kworker/u2:4-ext4-rsv-conversion) I 2 0 0 0 -1 69238880 0 2098 0 12 0 42 2 0 20 0 1 0 61 0 0 18446744073709551615 0 0 0 0 0 0 0 2147483647 0 1 0 0 17 0 0 0 0 0 0 0 0 0 0 0 0 0 0`)

	pinfo, err := ProcessStat(r)
	if err != nil {
		t.Fatalf("got %v, want `nil`", err)
	}

	if pinfo.PID != 97 {
		t.Errorf("got %v, want `97`", pinfo.PID)
	}

	if pinfo.Name != "(kworker/u2:4-ext4-rsv-conversion)" {
		t.Errorf("got %v, want `(kworker/u2:4-ext4-rsv-conversion)`", pinfo.Name)
	}

	if pinfo.IOWait != 0 {
		t.Errorf("got %v, want `0`", pinfo.IOWait)
	}
}

func generateProcFS(t *testing.T) string {
	d := t.TempDir()
	r, err := os.Open("stat_block.txt")
	if err != nil {
		panic(err)
	}
	defer r.Close()

	var ln string
	s := bufio.NewScanner(r)
	for s.Scan() {
		ln = s.Text()
		a := strings.SplitN(ln, " ", 2)

		p := path.Join(d, "proc", a[0])
		err := os.MkdirAll(p, 0700)
		if err != nil {
			panic(err)
		}
		w, err := os.Create(path.Join(p, "stat"))
		if err != nil {
			panic(err)
		}
		_, err = w.Write([]byte(ln + "\n"))
		if err != nil {
			panic(err)
		}
		w.Close()
	}

	return d
}
