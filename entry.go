package initmock

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"unsafe"
)

const (
	InitSkipFlag = "skip_pkgs"
)

type inittask struct {
	done  uintptr
	ndeps uintptr
	nfns  uintptr
}

func init() {
	skippkgs := readSkipPkgsInArgs()
	skipptrs, err := searchInitTaskByName(skippkgs)
	if err != nil {
		panic(err)
	}
	skip(skipptrs)
}

func readSkipPkgsInArgs() (out []string) {
	flag.String(InitSkipFlag, "", "the packages which skip their init function")
	return strings.Split(parseStringFlag(InitSkipFlag, os.Args[1:]), ",")
}

func parseStringFlag(flag string, args []string) string {
	pattern := fmt.Sprintf("-?-%s(=([^ ]+))?", flag)
	re := regexp.MustCompile(pattern)
	for i, arg := range args {
		if re.MatchString(arg) {
			subs := re.FindStringSubmatch(arg)
			if len(subs) > 2 && subs[2] != "" {
				// contain '='
				return subs[2]
			}
			// not contain '='
			if i+1 == len(args) {
				return ""
			}
			return args[i+1]
		}
	}
	return ""
}

func searchInitTaskByName(pkgnames []string) (ptrs []uintptr, err error) {
	if len(pkgnames) == 0 {
		return []uintptr{}, nil
	}
	var buf bytes.Buffer
	var syms []string
	var reexp string
	for _, pkgname := range pkgnames {
		sym := "_" + pkgname + "..inittask"
		syms = append(syms, sym)
	}
	reexp = strings.Join(syms, "\\|")

	shell := fmt.Sprintf("objdump -t %s | grep %s | awk -F' ' '{print $1}'", os.Args[0], reexp)
	cmd := exec.Command("bash", "-c", shell)
	cmd.Stdout = &buf
	if err = cmd.Run(); err != nil {
		return nil, fmt.Errorf("run cmd err=%v", err)
	}
	sc := bufio.NewScanner(&buf)
	for sc.Scan() {
		taskInt64, err := strconv.ParseInt(sc.Text(), 16, 64)
		if err != nil {
			return nil, fmt.Errorf("parse ptr err=%v", err)
		}
		ptrs = append(ptrs, uintptr(taskInt64))
	}
	return ptrs, nil
}

func skip(ptrs []uintptr) (err error) {
	for _, ptr := range ptrs {
		taskPtr := (*inittask)(unsafe.Pointer(ptr))
		taskPtr.done = 2
	}
	return nil
}
