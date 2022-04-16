package cmd

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"
	"regexp"
	"strings"

	"github.com/ghodss/yaml"
	"github.com/pelletier/go-toml"
)

func write(fields ...any) {
	fmt.Print(fields...)
}

func marshalout(o any, format formatMode) {
	write(string(marshal(o, format)))
}
func marshal(o any, format formatMode) []byte {
	switch format {
	case formatJson:
		j, err := json.MarshalIndent(o, "", "  ")
		if err != nil {
			l.Fatal().Err(err).Msg("failed to marshal (json)")
		}
		return j
	case formatToml:
		b := bytes.Buffer{}
		enc := toml.NewEncoder(&b)
		enc.CompactComments(true)
		enc.ArraysWithOneElementPerLine(true)
		enc.SetTagComment("help")
		enc.SetTagName("json")
		err := enc.Encode(o)
		if err != nil {
			l.Fatal().Err(err).Msg("failed to marshal (toml)")
		}
		return b.Bytes()
	}
	j, err := yaml.Marshal(o)
	if err != nil {
		l.Fatal().Err(err).Msg("failed to marshal (yaml)")
	}
	return j
}

func getFile(f string) (*os.File, bool) {
	if f == "" {
		return nil, false
	}

	if !path.IsAbs(f) {
		wd, err := os.Getwd()
		if err != nil {
			l.Fatal().Err(err).Str("path", f).Msg("Failed to get working directory for relative file")
		}
		f = path.Join(wd, f)
		l.Debug().Str("newpath", f).Msg("Rewrote relative path")

	}
	file, err := os.Open(f)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, false
		}
		l.Fatal().Err(err).Msg("Failed to open file")
	}
	return file, true
}

func runCmd(command string, fPath string, stdin io.Reader) ([]byte, error) {
	a := strings.Split(command, " ")
	cmd := a[0]
	args := []string{}
	if len(a) > 1 {
		args = a[1:]
	}
	args = append(args, fPath)
	c := exec.Command(cmd, args...)
	if stdin != nil {
		c.Stdin = stdin
	}
	if l.HasDebug() {
		l.Debug().
			Str("path", fPath).
			Str("cmd", cmd).
			Interface("args", args).
			Msg("Running command on replacement")
	}
	out, err := c.CombinedOutput()
	if err != nil {
		return out, fmt.Errorf("Failed to run onReplaceCmd %s %s %v: %w", c.Path, string(out), c.Args, err)
	}
	return out, nil
}
func commandExists(cmd string) bool {
	_, err := exec.LookPath(cmd)
	l.Debug().Str("cmd", cmd).Bool("found", err == nil).Msg("Checking for existance of command")
	return err == nil
}
func runPrettier(filepath string, contents io.Reader) ([]byte, error) {
	// TODO: also check yarn/npm/node-modules
	if commandExists(CLI.PrettierDSlimPath) {
		l.Debug().Msg("prettier_d_slim is available")
		if contents == nil {
			l.Debug().Str("path", filepath).Msg("Rereading from file")
			f, err := os.Open(filepath)
			if err != nil {
				return nil, fmt.Errorf("failed to read the file-contents prior to running command")
			}
			contents = f
			f.Close()
		}
		return runCmd(CLI.PrettierDSlimPath+" --stdin --stdin-filepath", filepath, contents)
	}
	if commandExists(CLI.PrettierPath) {
		l.Debug().Msg("prettier is available. Consider using prettier_d_slim if you want improved speed")
		return runCmd(CLI.PrettierPath+" -w --ignore-path NOEXIST", filepath, contents)
	}
	return nil, nil
}

func ReplaceAllStringSubmatchFunc(re *regexp.Regexp, str string, repl func([]string, int, int) string) string {
	result := ""
	lastIndex := 0

	for _, v := range re.FindAllSubmatchIndex([]byte(str), -1) {
		groups := []string{}
		for i := 0; i < len(v); i += 2 {
			groups = append(groups, str[v[i]:v[i+1]])
		}

		result += str[lastIndex:v[0]] + repl(groups, v[0], v[1])
		lastIndex = v[1]
	}

	return result + str[lastIndex:]
}
