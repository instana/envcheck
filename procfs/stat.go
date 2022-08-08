package procfs

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
	"regexp"
	"sort"
)

var re = regexp.MustCompile(`^[0-9]*$`)

// see proc(5) manpage.
const delayacct_blkio_ticks = 42

func ReadStats(p string) ([]*ProcessInfo, error) {
	proc := path.Join(p, "proc")
	info, err := ioutil.ReadDir(proc)
	if err != nil {
		return nil, err
	}

	var arr []*ProcessInfo
	for _, f := range info {
		if !f.IsDir() {
			continue
		}

		if re.FindString(f.Name()) != "" {
			r, err := os.Open(path.Join(proc, f.Name(), "stat"))
			if err != nil {
				return nil, err
			}
			pInfo, err := ProcessStat(r)
			if err != nil {
				return nil, err
			}
			r.Close()

			arr = append(arr, pInfo)
		}
	}

	sort.Sort(sort.Reverse(ByIOWait{arr}))

	return arr, nil
}

type ByPID []*ProcessInfo

func (s ByPID) Len() int           { return len(s) }
func (s ByPID) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }
func (s ByPID) Less(i, j int) bool { return s[i].PID < s[j].PID }

type ByIOWait struct{ ByPID }

func (s ByIOWait) Less(i, j int) bool { return s.ByPID[i].IOWait < s.ByPID[j].IOWait }

type ByName struct{ ByPID }

func (s ByName) Less(i, j int) bool { return s.ByPID[i].Name < s.ByPID[j].Name }

func ProcessStat(r io.Reader) (*ProcessInfo, error) {
	var discard string
	var pinfo = ProcessInfo{
		IOWait: -1,
	}

	_, err := fmt.Fscanf(r, "%d %s %s", &pinfo.PID, &pinfo.Name, &discard)
	if err != nil {
		return nil, err
	}

	col := 3
	for {
		_, err = fmt.Fscanf(r, "%f", &pinfo.IOWait)
		if err != nil {
			return nil, err
		}
		col++
		if col == delayacct_blkio_ticks {
			break
		}
	}
	return &pinfo, nil
}

type ProcessInfo struct {
	PID    int
	Name   string
	IOWait float64
}
