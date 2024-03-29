# init mock

## Motivation

The `init` function is a powerful but dangerous language feature in golang.
Some annoying logics hidden in `init` functions and make it difficult to do unit tests.
So I make a lightweight library to skip them.

Replacing the init function is a little bit tricky. At first I wanted to find a way to mock init function with monkey-patching,
but it's not easy to control the package initialization order. You have to init a special package before the package that you
want to mock. It's also difficult to hack the go compiler. I think the easiest way to do so is to do it after compiling and
before the execution. Luckily go test have the `exec` flag to pass the compiled executable file in commandline to a specific program.


## Fundamentals

* How a golang program compiles and calls init function

* The executable file (elf) layout

* How do we locate the inittask in executable files?

    * symbol name => VMA
    * data Section VMA + data Section File Offset - VMA => symbol file offset
    * symbol file offset -> init done flag + init function pointer

* How to skip package init?

    * set inittask state to 2

* How to replace init function?

    * locate the both init task;
    * skip the source init function;
    * swap the source and destination init function pointer in each task;

## Usage

1. install initmock: 

```
go install github.com/huiscool/initmock
```

2. run go test with `-exec initmock` build flag and addition binary flags:
`-skippkg=[package name]`: skip the package initialization
`-skipinit=[init func name]`: skip the init function execution
`-replaceinit=[src init func name]:[dst init func name]`: replace init function with another one

NOTICE 1: 
IMHO build/test flags are passed to go and binary flags are passed to test binary. `initmock` read the
`skipinit` and `replaceinit` from the binary flags, which settles after the package arguments. Unkwown
build/test flags passed to go-test will prevent running tests.

NOTICE 2:
To skip/replace multiple init functions, use multiple `skipinit`/`replaceinit` flag;
`initmock` will extract the `skipinit`/`replaceinit` flag with below regular expression:
`-?-(skipinit|replaceinit)[= ](\S+)?`

NOTICE 3:
`go test` may omit debug info when compile flag is not set. Use flags below to keep test binary:
`-blockprofile` `-cpuprofile` `-memprofile` `-mutexprofile` `-c` `-o`

NOTICE 4:
Initmock assumes the executable file's arch and os is the same with itself. Initmock doesn't work with cross-compiling.
Currently we only support `windows/amd64` , `darwin/amd64`, `linux/amd64`

example:
```
go test ./testmain -exec initmock -v -skippkg github.com/huiscool/initmock/testmain/panic
go test ./testmain -exec initmock -v -skipinit github.com/huiscool/initmock/testmain/panic.init.0
go test ./testmain -exec initmock -v -replaceinit github.com/huiscool/initmock/testmain/panic.init.0:github.com/huiscool/initmock/testmain.init.0
```

## Reference

[1] [【Go】一文带你深入了解初始化函数](https://juejin.cn/post/7011737366360490015)