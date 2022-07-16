package main

import (
	"debug/macho"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"syscall"
)

func main() {
	if len(os.Args) < 2 {
		help()
		return
	}
	execName := os.Args[1]
	args := os.Args[2:]
	toSkip, toReplace, rest := extractArgs(args)
	if len(toSkip) > 0 {
		skip(execName, toSkip)
	}
	if len(toReplace) > 0 {
		replace(execName, toReplace)
	}
	syscall.Exec(execName, rest, os.Environ())
}

//==============================================================================
// flags & prompts
//==============================================================================

const (
	SkipFlag    = "skipinit"
	ReplaceFlag = "replaceinit"
)

var helpPrompt = `
init mock
usage: 
go test ./testmain -exec initmock -v -skipinit github.com/testmain/panic
`

func help() {
	fmt.Println(helpPrompt)
}

func extractArgs(args []string) (skipped []string, replaced []string, rest []string) {
	copied := make([]string, len(args))
	copy(copied, args)
	var argmap = map[string][]string{
		SkipFlag:    {},
		ReplaceFlag: {},
	}
	for i := 0; i < len(args); i++ {
		subs := regexp.MustCompile(`^-?-(skipinit|replaceinit)(=([^\s-]+))?$`).FindStringSubmatch(args[i])
		if subs == nil {
			rest = append(rest, args[i])
			continue
		}
		// is skipinit/replaceinit flag
		// try to get argument after =
		flagName := subs[1]
		expWithEq := subs[2]
		exp := subs[3]
		if expWithEq != "" {
			argmap[flagName] = append(argmap[flagName], exp)
			continue
		}
		// look ahead for the space separated arg
		i++
		if i < len(args) && regexp.MustCompile(`^[^\s-]+$`).MatchString(args[i]) {
			argmap[flagName] = append(argmap[flagName], args[i])
			continue
		}
		i--
		// illegal skip/replace flag, throw them in rest
		rest = append(rest, args[i])
	}
	return argmap[SkipFlag], argmap[ReplaceFlag], rest
}

//==============================================================================
// service
//==============================================================================

func skip(execName string, toSkips []string) {
	fmt.Println(execName)
	c := openMacho(execName)
	defer c.close()
	// sym := c.getSym("github.com/huiscool/initmock/testmain/panic..inittask")
	// cmd := exec.Command("objdump", "-t", execName)
	// cmd.Stdout = os.Stdout
	// cmd.Stderr = os.Stderr
	// err := cmd.Run()
	// mayExitOn(err)
	return
}

func replace(execName string, toReplaces []string) {
	return
}

//==============================================================================
// helpers
//==============================================================================

func mayExitOn(err error) {
	if err != nil {
		panic(err)
	}
}

//==============================================================================
// binary reader
//==============================================================================
type Platform int

const (
	Linux   Platform = 1
	Darwin  Platform = 2
	Windows Platform = 3
)

type goexec interface {
	file() *os.File
	getInitTask(pkgname string) *initTask
}

type initTask struct {
	exec       goexec
	name       string
	vmOffset   uintptr
	fileOffset uintptr
	infile     *initTaskInFile
}

func (t *initTask) save() {
	raw := []byte{}
	f := t.exec.file()
	_, err := f.WriteAt(raw, int64(t.fileOffset))
	mayExitOn(err)
}

type initTaskInFile struct {
	status uintptr
	ndeps  uintptr
	nfns   uintptr
	deps   []uintptr
	fns    []uintptr
}

type initFunc struct {
	name       string
	vmOffset   uintptr
	fileOffset uintptr
}

type machoExec struct {
	f     *os.File
	macho *macho.File
}

var _ goexec = (*machoExec)(nil)

func openMacho(fname string) (exec *machoExec) {
	f, err := os.OpenFile(fname, os.O_RDWR, os.ModePerm)
	mayExitOn(err)
	macho, err := macho.NewFile(f)
	return &machoExec{
		f:     f,
		macho: macho,
	}
}

func (m *machoExec) file() *os.File {
	return m.f
}

func (m *machoExec) getInitTask(pkgName string) *initTask {
	return nil
}

func pretty(obj interface{}) string {
	bin, _ := json.MarshalIndent(obj, "", "  ")
	return string(bin)
}
