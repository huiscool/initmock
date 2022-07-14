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

* The executable file (elf) layout

* How a golang program compiles and calls init function

## Usage

1. install initmock: 

```
go install github.com/huiscool/initmock
```

2. run go test with `-exec initmock` build flag and addition binary flags:
`-skipinit [package name]`: skip the flag
~~`-replaceinit [src init func name],[dst init func name]`: replace init function with another one~~

NOTICE 1: 
IMHO build/test flags are passed to go and binary flags are passed to test binary. `initmock` read the
`skipinit` and ~~`replaceinit`~~ from the binary flags, which settles after the package arguments. Unkwown
build/test flags passed to go-test will prevent running tests.

NOTICE 2:
To skip/replace multiple packages, use multiple `skipinit`/`replaceinit` flag;
`initmock` will extract the `skipinit`/`replaceinit` flag with below regular expression:
`-?-(skipinit|replaceinit)[= ](\S+)?`

example:
```
go test ./testmain -exec initmock -v -skipinit github.com/testmain/panic
go test ./testmain -exec initmock -v -skipinit github.com/testmain/panic_one -skipinit 
```

## Reference

[1] [【Go】一文带你深入了解初始化函数](https://juejin.cn/post/7011737366360490015)