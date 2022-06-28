# init mock

## Motivation

The `init` function is a powerful but dangerous language feature in golang.
Some annoying logics hidden in `init` functions and make it difficult to do unit tests.
So I make a lightweight library to skip them.

## Fundamentals

* The executable file (elf) layout

* How a golang program compiles and calls init function

* The package init order in golang

## Usage

1. `go get github.com/huiscool/initmock` in your golang project.

2. import initmock in your test file like this:
```
import _ "github.com/huiscool/initmock"
```

3. pass the package names (comma seperated) in `--skip_pkgs` flag when testing, just like this:
```
go test ./test --skip_pkgs github.com/huiscool/initmock/test/panic

// output:
=== RUN   TestMain
--- PASS: TestMain (0.00s)
PASS
ok      github.com/huiscool/initmock/test       0.669s
```

## Limitation

* Cannot skip standard library, because initmock rely on them.

* It is not easy to control the package init order precisely.

## Reference

[1] [【Go】一文带你深入了解初始化函数](https://juejin.cn/post/7011737366360490015)