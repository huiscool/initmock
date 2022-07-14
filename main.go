package main

import (
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
	fmt.Println(string([]byte{207, 250, 237, 254}))
	ctx := open(execName)
	defer ctx.close()
	// cmd := exec.Command("readelf", "--symbols", execName)
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

type binCtx struct {
}

func open(fname string) (ctx *binCtx) {
	panic("not implemented")
}

func (c *binCtx) getSyms(symName string) {
	panic("not implemented")
}

func (c *binCtx) getInitTask(pkgName string) (*inittask, error) {
	panic("not implemented")
}

func (c *binCtx) getInitFunc(funcName string) (*initfunc, error) {
	panic("not implemented")
}

func (c *binCtx) close() {
	panic("not implemented")
}

type inittask struct {
	fileOffset int
	menOffset  int
	name       string
}

type initfunc struct {
	fileOffset int
	menOffset  int
	name       string
}
