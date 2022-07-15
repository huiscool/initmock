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
	c := open(execName)
	defer c.close()
	sym := c.getSym("github.com/huiscool/initmock/testmain/panic..inittask")
	fmt.Println(pretty(sym))
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
	open(fname string)
	close()
	getInitTask(pkgname string) *initTask
	getInitFunc(funcname string) *initFunc
}

type initTask struct{}
type initFunc struct{}

type machoExec struct {
	f *macho.File
}

func open(fname string) (exec *machoExec) {
	f, err := macho.Open(fname)
	mayExitOn(err)
	return &machoExec{f: f}
}

func (m *machoExec) getSym(name string) *macho.Symbol {
	for i := range m.f.Symtab.Syms {
		sym := m.f.Symtab.Syms[i]
		if sym.Name == name {
			return &sym
		}
	}
	return nil
}

func (c *machoExec) close() {
	c.f.Close()
}

func pretty(obj interface{}) string {
	bin, _ := json.MarshalIndent(obj, "", "  ")
	return string(bin)
}
