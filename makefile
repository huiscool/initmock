
testm:
	rm testm
	go test -o testm -c ./testmain
	go tool objdump ./testm > testm.objdump

skip: 
	go install .
	initmock ./testm -skipinit github.com/huiscool/initmock/testmain/panic.init.0 -v

replace: 
	go install .
	initmock ./testm -replaceinit github.com/huiscool/initmock/testmain/panic.init.0:github.com/huiscool/initmock/testmain.init.0 -v

.PHONY: testm replace skip