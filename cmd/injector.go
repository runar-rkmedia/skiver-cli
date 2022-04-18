package cmd

import (
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/runar-rkmedia/go-common/logger"
)

type Injecter struct {
	l               logger.AppLogger
	Dir             string
	DryRun          bool
	OnReplaceCmd    string
	ExtensionFilter map[string]bool
	IgnoreFilter    []string
	Regex           *regexp.Regexp
	ReplacementFunc ReplacementFunc
}

type ReplacementFunc = func(groups []string) (s string, changed bool)

type st = struct {
	FilePath string
	Start    time.Time
	Duration time.Duration
}
type _written []st

func (st _written) Len() int {
	return len(st)
}
func (st _written) Less(i, j int) bool {
	return st[i].Duration < st[j].Duration
}
func (st _written) Swap(i, j int) {
	st[i], st[j] = st[j], st[i]
}

func NewInjector(l logger.AppLogger, dir string, dryRun bool, onReplace string, ignoreFilter []string, extFilter []string, regex *regexp.Regexp, replacementFunc ReplacementFunc) Injecter {

	in := Injecter{
		l:               l,
		DryRun:          dryRun,
		Dir:             dir,
		OnReplaceCmd:    onReplace,
		IgnoreFilter:    ignoreFilter,
		ExtensionFilter: map[string]bool{},
		Regex:           regex,
		ReplacementFunc: replacementFunc,
	}

	for _, ext := range extFilter {
		in.ExtensionFilter[ext] = true
	}

	return in
}

func (in Injecter) Inject() error {
	in.l.Debug().
		Str("dir", in.Dir).
		Msg("Started injection in path")
	paths := map[string]fs.FileInfo{}
	wg := sync.WaitGroup{}

	type s = struct {
		FilePath string
		Info     fs.FileInfo
	}
	concurrency := runtime.NumCPU()
	ch := make(chan s, concurrency)
	writtenCh := make(chan st)

	var written _written

	start := time.Now()

	for i := 0; i < concurrency; i++ {
		go func() error {
			for {
				select {
				case ss := <-ch:
					sst := st{FilePath: ss.FilePath, Start: time.Now()}
					changed, err := in.VisitFile(ss.FilePath, ss.Info)
					sst.Duration = time.Now().Sub(sst.Start)
					if err != nil {
						in.l.Fatal().Err(err).Str("path", ss.FilePath).Msg("Failed replacement in file")
					}
					if in.l.HasDebug() {
						in.l.Debug().
							Str("path", sst.FilePath).
							Str("duration", sst.Duration.String()).
							Bool("changed", changed).
							Msg("Completed replacement in file")
					}
					if changed {
						writtenCh <- sst
					}
					wg.Done()
				}
			}
		}()
	}

	go func() {
		for {
			select {
			case sst := <-writtenCh:
				written = append(written, sst)
			}
		}
	}()

	var walker filepath.WalkFunc = func(fPath string, info fs.FileInfo, err error) error {
		if info == nil {
			return fmt.Errorf("fileInfo was nil for %s", fPath)
		}
		if info.IsDir() {
			return nil
		}
		name := info.Name()
		ext := strings.TrimPrefix(path.Ext(name), ".")
		if _, ok := in.ExtensionFilter[ext]; !ok {
			return nil
		}
		for _, ignore := range in.IgnoreFilter {
			// TODO: use gitignore etc.
			if strings.Contains(name, ignore) {
				return nil
			}

		}
		paths[fPath] = info
		ch <- s{fPath, info}
		wg.Add(1)
		return nil
	}
	if err := filepath.Walk(in.Dir, walker); err != nil {
		return err
	}
	debug := in.l.HasDebug()
	if debug {
		in.l.Debug().
			Str("dir", in.Dir).
			Int("count", len(paths)).
			Int("concurrency", concurrency).
			Msg("Started injection for files")

	}
	wg.Wait()

	if debug {
		in.l.Debug().
			Str("dir", in.Dir).
			Int("count", len(paths)).
			Int("concurrency", concurrency).
			Int("writtenCount", len(written)).
			Str("duration", time.Now().Sub(start).String()).
			Msg("Completed injection for files")
	}

	if debug {
		sort.Sort(written)
		for _, v := range written {
			fmt.Println(v.Duration, v.FilePath)
		}
	}

	return nil

}
func (in Injecter) VisitFile(fPath string, info fs.FileInfo) (bool, error) {
	l := logger.With(in.l.With().Str("dir", in.Dir).Logger())

	f, b, err := GetFileAndContent(in.DryRun, fPath, info)
	if err != nil {
		return false, fmt.Errorf("Failed to read file %s: %w", fPath, err)
	}
	s := string(b)
	changed := false
	replacement := ReplaceAllStringSubmatchFunc(in.Regex, s, func(groups []string, start, end int) string {
		if len(groups) < 4 {
			l.Fatal().Interface("groups", groups).Msg("Expected to have 4 groups (whole match, prefix, content and suffix)")
		}
		repl, hasChange := in.ReplacementFunc(groups[1:])
		if !hasChange {
			return groups[0]
		}
		changed = true
		return repl
	})

	if !changed {
		return false, nil
	}

	if in.DryRun {
		fmt.Println(fPath)
		fmt.Println(replacement)
		return false, nil
	}
	f.Seek(0, 0)
	n, err := f.Write([]byte(replacement))
	if err != nil {
		return true, fmt.Errorf("Error writing replacement to file '%s': %s",
			fPath, err)
	}
	if int64(n) < info.Size() {
		err := f.Truncate(int64(n))
		if err != nil {
			return true, fmt.Errorf("Error truncating file '%s' to size %d",
				fPath, n)
		}
	}
	if in.OnReplaceCmd != "" {
		if _, err := runCmd(in.OnReplaceCmd, fPath, strings.NewReader(s)); err != nil {
			return true, err
		}
	}
	return true, nil
}

func GetFileAndContent(dryRun bool, fn string, fi os.FileInfo) (f *os.File, content []byte, err error) {

	if dryRun {
		f, err = os.Open(fn)
	} else {
		f, err = os.OpenFile(fn, os.O_RDWR, 0666)
	}

	if err != nil {
		return
	}

	content = make([]byte, fi.Size())
	n, err := f.Read(content)
	if err != nil {
		return
	}
	if int64(n) != fi.Size() {
		err = fmt.Errorf("Thw whole file was not read, only %s of %s", humanize.Bytes(uint64(n)), humanize.Bytes(uint64(fi.Size())))
	}

	return
}
