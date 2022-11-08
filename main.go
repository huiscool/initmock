package main

import (
	"debug/elf"
	"debug/macho"
	"debug/pe"
	"fmt"
	"io"
	"os"
	"reflect"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"unsafe"
)

var DebugFlag = true

func main() {
	if len(os.Args) < 2 {
		help()
		return
	}
	debug("args=%v", os.Args)
	execName := os.Args[1]
	args := os.Args[2:]
	argmap, rest := extractArgs(args, []string{
		ReplaceInitFlag,
		SkipInitFlag,
		SkipPkgFlag,
	})
	if len(argmap[SkipPkgFlag]) > 0 {
		skipPkg(execName, argmap[SkipPkgFlag])
	}
	if len(argmap[SkipInitFlag]) > 0 {
		skipInit(execName, argmap[SkipInitFlag])
	}
	if len(argmap[ReplaceInitFlag]) > 0 {
		replaceInit(execName, argmap[ReplaceInitFlag])
	}
	syscall.Exec(execName, rest, os.Environ())
}

//==============================================================================
// flags & prompts
//==============================================================================

const (
	SkipPkgFlag     = "skippkg"
	SkipInitFlag    = "skipinit"
	ReplaceInitFlag = "replaceinit"
)

var helpPrompt = `
init mock
usage: 
go test ./testmain -exec initmock -v -skippkg github.com/huiscool/initmock/testmain/panic
go test ./testmain -exec initmock -v -skipinit github.com/huiscool/initmock/testmain/panic.init.0
go test ./testmain -exec initmock -v -replaceinit github.com/huiscool/initmock/testmain/panic.init.0:github.com/huiscool/initmock/testmain.init.0
`

func help() {
	fmt.Println(helpPrompt)
}

func extractArgs(args []string, flags []string) (argmap map[string][]string, rest []string) {
	copied := make([]string, len(args))
	copy(copied, args)
	argmap = map[string][]string{}
	// check flag name
	for i := range flags {
		if !regexp.MustCompile(`^[a-zA-Z]+$`).MatchString(flags[i]) {
			Exitf("invalid flag (%s)", flags[i])
		}
	}
	re := regexp.MustCompile(fmt.Sprintf(`^-?-(%s)(=([^\s-]+))?$`, strings.Join(flags, "|")))
	for i := 0; i < len(args); i++ {
		subs := re.FindStringSubmatch(args[i])
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
	return argmap, rest
}

//==============================================================================
// service
//==============================================================================

func skipPkg(execName string, toSkips []string) {
	exec := open(execName)
	defer exec.file().Close()
	for _, toSkip := range toSkips {
		task := exec.getInitTask(toSkip)
		task.infile.status = 2
		writeInitTaskAt(exec.file(), int(task.fileOffset), task.infile)
	}
}

func skipInit(execName string, toSkips []string) {
	exec := open(execName)
	defer exec.file().Close()
	for _, raw := range toSkips {
		f := exec.getInitFunc(raw)
		skipInitFunc(exec.file(), f)
	}
}

func replaceInit(execName string, toReplaces []string) {
	srcToDst := parseAndCheckReplaceArgs(toReplaces)
	exec := open(execName)
	defer exec.file().Close()
	for src, dst := range srcToDst {
		doReplace(exec, src, dst)
	}
}

// 1. invalidate src function.
// 2. swap function pointer in src/dst inittask;
func doReplace(exec goexec, srcRaw, dstRaw string) {
	src := exec.getInitFunc(srcRaw)
	dst := exec.getInitFunc(dstRaw)
	srcTask := exec.getInitTask(src.pkgName)
	dstTask := exec.getInitTask(dst.pkgName)
	debug("src=%+v, srctask=%+v", src, srcTask.infile)
	debug("dst=%+v, dsttask=%+v", dst, dstTask.infile)
	skipInitFunc(exec.file(), src)
	getFnIdxInTask := func(t *initTask, f *initFunc) int {
		for i, fptr := range t.infile.fns {
			if fptr == uintptr(f.vmoffset) {
				return i
			}
		}
		Exitf("cannot find function %s in inittask", f.initFuncName.Raw())
		return -1
	}
	si := getFnIdxInTask(srcTask, src)
	di := getFnIdxInTask(dstTask, dst)
	sptr := srcTask.infile.fns[si]
	dptr := dstTask.infile.fns[di]
	srcTask.infile.fns[si] = dptr
	dstTask.infile.fns[di] = sptr
	// write back
	writeInitTaskAt(exec.file(), int(srcTask.fileOffset), srcTask.infile)
	writeInitTaskAt(exec.file(), int(dstTask.fileOffset), dstTask.infile)
}

func parseAndCheckReplaceArgs(toReplaces []string) map[string]string {
	var srcToDst = map[string]string{}
	// further parse args
	for _, item := range toReplaces {
		if item == "" {
			Exitf("empty replace argument")
		}
		funcNames := strings.Split(item, ":")
		if len(funcNames) != 2 {
			Exitf("invalid replace argument %s, should be like \"<source init function name>:<destination init function name>\"", item)
		}
		src, dst := funcNames[0], funcNames[1]
		if another, ok := srcToDst[src]; ok {
			Exitf("invalid replace argument %s, conflicts with %s", item, src+":"+another)
		}
		srcToDst[src] = dst
	}
	// check func form
	for src, dst := range srcToDst {
		initFuncNameFromRaw(src)
		initFuncNameFromRaw(dst)
	}
	// for now we only support one step replace.
	for _, dst := range srcToDst {
		if _, ok := srcToDst[dst]; ok {
			Exitf("invalid replace argument, cannot replace destination function %s", dst)
		}
	}
	return srcToDst
}

//==============================================================================
// helpers
//==============================================================================

func Exitf(format string, args ...interface{}) {
	panic(fmt.Sprintf(format, args...))
}

func mayExitOn(err error, args ...interface{}) {
	if err != nil {
		if len(args) > 0 {
			fmtStr := fmt.Sprintf(args[0].(string), args[1:]...)
			err = fmt.Errorf("%s:%w", fmtStr, err)
		}
		panic(err)
	}
}

func open(name string) (exec goexec) {
	dist := fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH)
	switch dist {
	case "linux/amd64":
		return openElf(name)
	case "windows/amd64":
		return openPE(name)
	case "darwin/amd64":
		return openMacho(name)
	default:
		Exitf("unknown distribution %s", dist)
		return nil
	}
}

func debug(format string, arg ...interface{}) {
	if DebugFlag {
		fmt.Printf(format, arg...)
		fmt.Println()
	}
}

//==============================================================================
// abstract exec
//==============================================================================

type goexec interface {
	file() *os.File
	getInitTask(pkgname string) *initTask
	getInitFunc(funcname string) *initFunc
}

type symbol struct {
	name     string
	vmoffset uint64
}

type sectionInfo struct {
	sectname   string
	segname    string
	vmoffset   uint64
	fileoffset uint64
}

type initTask struct {
	name       string
	vmoffset   uint64
	fileOffset uint64
	infile     *initTaskInFile
}

type initFunc struct {
	initFuncName
	vmoffset   uint64
	fileOffset uint64
}

type initFuncName struct {
	pkgName string
	index   int32
}

func (f initFuncName) Raw() string {
	return fmt.Sprintf("%s.init.%d", f.pkgName, f.index)
}

func (f initFuncName) TaskName() string {
	return taskNameFromRawFuncname(f.Raw())
}

type initTaskHeader struct {
	status uintptr
	ndeps  uintptr
	nfns   uintptr
}

type initTaskInFile struct {
	initTaskHeader
	deps []uintptr
	fns  []uintptr
}

func readInitTaskAt(f io.ReaderAt, fileOffset uint64) *initTaskInFile {
	// TODO: handle non-64 platform
	const ptrsize = unsafe.Sizeof(uintptr(0))
	const headerSize = unsafe.Sizeof(initTaskHeader{})
	// read header
	var err error
	var headerbin = make([]byte, headerSize)
	_, err = f.ReadAt(headerbin, int64(fileOffset))
	mayExitOn(err, "cannot read inittask header")

	header := **(**initTaskHeader)(unsafe.Pointer(&headerbin))

	out := &initTaskInFile{
		initTaskHeader: header,
		deps:           []uintptr{},
		fns:            []uintptr{},
	}
	// read deps and fns
	var bin = make([]byte, ptrsize*(header.ndeps+header.nfns))

	_, err = f.ReadAt(bin, int64(uintptr(fileOffset)+headerSize))
	mayExitOn(err, "cannot read inittask")
	for i := 0; i < int(header.ndeps); i++ {
		ptrbin := bin[i*int(ptrsize) : (i+1)*int(ptrsize)]
		out.deps = append(out.deps,
			**(**uintptr)(unsafe.Pointer(&ptrbin)),
		)
	}
	bin = bin[ptrsize*header.ndeps:]
	for i := 0; i < int(header.nfns); i++ {
		ptrbin := bin[i*int(ptrsize) : (i+1)*int(ptrsize)]
		out.fns = append(out.fns,
			**(**uintptr)(unsafe.Pointer(&ptrbin)),
		)
	}
	return out
}

func writeInitTaskAt(f io.WriterAt, fileOffset int, task *initTaskInFile) {
	out := cancat([][]byte{
		ptrToBin(&task.status),
		ptrToBin(&task.ndeps),
		ptrToBin(&task.nfns),
		ptrsToBin(task.deps),
		ptrsToBin(task.fns),
	}...)
	debug("0x%x: write %v", fileOffset, out)
	_, err := f.WriteAt(out, int64(fileOffset))
	mayExitOn(err, "write init task")
}

func cancat(bins ...[]byte) []byte {
	var out []byte
	for i := range bins {
		out = append(out, bins[i]...)
	}
	return out
}

func ptrToBin(p *uintptr) []byte {
	bin := *(*[unsafe.Sizeof(uintptr(0))]byte)(unsafe.Pointer(p))
	return bin[:]
}
func ptrsToBin(p []uintptr) []byte {
	ptrSize := unsafe.Sizeof(uintptr(0))
	h := (*reflect.SliceHeader)(unsafe.Pointer(&p))
	binh := reflect.SliceHeader{
		Data: h.Data,
		Len:  h.Len * int(ptrSize),
		Cap:  h.Cap * int(ptrSize),
	}
	bin := *(*[]byte)(unsafe.Pointer(&binh))
	return bin
}

func genInittask(
	symName string,
	sectName string,
	syms map[string]*symbol,
	sects map[string]*sectionInfo,
	file io.ReaderAt,
) *initTask {

	sym, ok := syms[symName]
	if !ok {
		Exitf("cannot find %s in symbol table", symName)
	}
	// get file offset from sections
	sect, ok := sects[sectName]
	if !ok {
		Exitf("cannot find data section info")
	}
	foffset := sect.fileoffset + (sym.vmoffset - sect.vmoffset)
	infile := readInitTaskAt(file, foffset)
	debug("0x%x: read %s: %+v", foffset, symName, infile)

	return &initTask{
		name:       symName,
		vmoffset:   sym.vmoffset,
		fileOffset: foffset,
		infile:     infile,
	}
}

func genInitFunc(
	symName string,
	sectName string,
	syms map[string]*symbol,
	sects map[string]*sectionInfo,
	file io.ReaderAt,
) *initFunc {
	sym, ok := syms[symName]
	if !ok {
		Exitf("cannot find %s in symbol table", symName)
	}
	// get file offset from sections
	sect, ok := sects[sectName]
	if !ok {
		Exitf("cannot find %s section info", sectName)
	}
	foffset := sect.fileoffset + (sym.vmoffset - sect.vmoffset)
	return &initFunc{
		initFuncName: initFuncNameFromRaw(symName),
		vmoffset:     sym.vmoffset,
		fileOffset:   foffset,
	}
}

func writeInitFuncAt(f io.WriterAt, fileOffset int, bin []byte) {
	_, err := f.WriteAt(bin, int64(fileOffset))
	mayExitOn(err, "cannot write init func")
}

func skipInitFunc(f io.WriterAt, initfunc *initFunc) {
	var ret = []byte{0xC3} // x86 return instruction
	writeInitFuncAt(f, int(initfunc.fileOffset), ret)
}

func taskNameFromRawFuncname(funcname string) string {
	f := initFuncNameFromRaw(funcname)
	return fmt.Sprintf("%s..inittask", f.pkgName)
}

func initFuncNameFromRaw(raw string) initFuncName {
	submatch := regexp.MustCompile(`^(\S+)\.init\.(\d+)$`).FindStringSubmatch(raw)
	if submatch == nil {
		Exitf("failed to extract init func name %s", raw)
	}
	index, err := strconv.ParseInt(submatch[2], 10, 0)
	mayExitOn(err, "failed to parse init func index")
	return initFuncName{
		pkgName: submatch[1],
		index:   int32(index),
	}
}

//==============================================================================
// macho
//==============================================================================

type machoExec struct {
	f     *os.File
	macho *macho.File
	syms  map[string]*symbol
	sects map[string]*sectionInfo
}

var _ goexec = (*machoExec)(nil)

func openMacho(fname string) (exec *machoExec) {
	f, err := os.OpenFile(fname, os.O_RDWR, os.ModePerm)
	mayExitOn(err)
	machoFile, err := macho.NewFile(f)
	mayExitOn(err)
	out := &machoExec{
		f:     f,
		macho: machoFile,
	}
	out.genSyms()
	out.genSectInfos()
	return out
}

func (m *machoExec) genSyms() {
	syms := map[string]*symbol{}
	for i := range m.macho.Symtab.Syms {
		sym := &m.macho.Symtab.Syms[i]
		syms[sym.Name] = &symbol{
			name:     sym.Name,
			vmoffset: sym.Value,
		}
		debug("load syms: %s,val=0x%x", sym.Name, sym.Value)
	}
	m.syms = syms

}

func (m *machoExec) genSectInfos() {
	// read section load commands
	sects := map[string]*sectionInfo{}
	for i := range m.macho.Sections {
		sect := m.macho.Sections[i]
		sects[sect.Name] = &sectionInfo{
			sectname:   sect.Name,
			segname:    sect.Seg,
			vmoffset:   sect.Addr,
			fileoffset: uint64(sect.Offset),
		}
		debug("load section: %s,%s", sect.Seg, sect.Name)
	}
	m.sects = sects
}

func (m *machoExec) file() *os.File {
	return m.f
}

func (m *machoExec) getInitTask(pkgName string) *initTask {
	symName := fmt.Sprintf("%s..inittask", pkgName)
	sectName := "__noptrdata"
	return genInittask(symName, sectName, m.syms, m.sects, m.f)
}

func (m *machoExec) getInitFunc(funcName string) *initFunc {
	symName := funcName
	sectName := "__text"
	out := genInitFunc(symName, sectName, m.syms, m.sects, m.f)
	return out
}

//==============================================================================
// elf
//==============================================================================

type elfExec struct {
	f     *os.File
	elf   *elf.File
	syms  map[string]*symbol
	sects map[string]*sectionInfo
}

func openElf(fname string) *elfExec {
	f, err := os.OpenFile(fname, os.O_RDWR, os.ModePerm)
	mayExitOn(err)
	elffile, err := elf.NewFile(f)
	mayExitOn(err)
	out := &elfExec{
		f:   f,
		elf: elffile,
	}
	out.genSectInfos()
	out.genSyms()
	return out
}

func (e *elfExec) genSyms() {
	syms := map[string]*symbol{}
	symbols, err := e.elf.Symbols()
	mayExitOn(err, "gen syms")
	for i := range symbols {
		sym := &symbols[i]
		syms[sym.Name] = &symbol{
			name:     sym.Name,
			vmoffset: sym.Value,
		}
		debug("load syms: %s,val=0x%x", sym.Name, sym.Value)
	}
	e.syms = syms
}

func (e *elfExec) genSectInfos() {
	// read sections
	sects := map[string]*sectionInfo{}
	for i := range e.elf.Sections {
		sect := e.elf.Sections[i]
		sects[sect.Name] = &sectionInfo{
			sectname:   sect.Name,
			segname:    "",
			vmoffset:   sect.Addr,
			fileoffset: uint64(sect.Offset),
		}
		debug("load section: %s", sect.Name)
	}
	e.sects = sects
}

func (e *elfExec) file() *os.File {
	return e.f
}

func (e *elfExec) getInitTask(pkgName string) *initTask {
	symName := fmt.Sprintf("%s..inittask", pkgName)
	sectName := ".noptrdata"
	return genInittask(symName, sectName, e.syms, e.sects, e.f)
}

func (e *elfExec) getInitFunc(funcName string) *initFunc {
	symName := funcName
	sectName := ".text"
	return genInitFunc(symName, sectName, e.syms, e.sects, e.f)
}

//==============================================================================
// pe
//==============================================================================

type peExec struct {
	f     *os.File
	pe    *pe.File
	syms  map[string]*symbol
	sects map[string]*sectionInfo
}

func openPE(fname string) *peExec {
	f, err := os.OpenFile(fname, os.O_RDWR, os.ModePerm)
	mayExitOn(err)
	pefile, err := pe.NewFile(f)
	mayExitOn(err)
	out := &peExec{
		f:  f,
		pe: pefile,
	}
	return out
}

func (p *peExec) genSyms() {
	syms := map[string]*symbol{}
	symbols := p.pe.Symbols
	for i := range symbols {
		sym := symbols[i]
		syms[sym.Name] = &symbol{
			name:     sym.Name,
			vmoffset: uint64(sym.Value),
		}
		// when storage class = 3,
		// value represents the offset inside a section
		debug("load syms: %s,val=0x%x sect=%d type=%d", sym.Name, sym.Value, sym.SectionNumber, sym.StorageClass)
	}
	p.syms = syms
}

func (p *peExec) genSectInfos() {
	// read sections
	sects := map[string]*sectionInfo{}
	for i := range p.pe.Sections {
		sect := p.pe.Sections[i]
		sects[sect.Name] = &sectionInfo{
			sectname: sect.Name,
			segname:  "",
			vmoffset: 0,
			// As (sym.vmoffset - sect.vmoffset) should be the section offset,
			// and sym.vmoffset is also the section offset in pe file,
			// We can set sect.vmoffset to zero here.
			fileoffset: uint64(sect.Offset),
		}
		debug("load section: %s", sect.Name)
	}
	p.sects = sects
}

func (p *peExec) file() *os.File {
	return p.f
}

func (p *peExec) getInitTask(pkgName string) *initTask {
	symName := fmt.Sprintf("%s..inittask", pkgName)
	sectName := ".data"
	return genInittask(symName, sectName, p.syms, p.sects, p.f)
}

func (p *peExec) getInitFunc(funcName string) *initFunc {
	symName := funcName
	sectName := ".text"
	return genInitFunc(symName, sectName, p.syms, p.sects, p.f)
}
