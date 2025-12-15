#@author Fred Brooker <git@gscloud.cz>

all:
	@echo "build | run | test | everything";

build:
	@echo "Building app ..."
	@cd go/ && go build -o logcleaner .

test: build
	@cd go/ && go test -v .

realtest: build
	@cp go/test_log.txt go/realtest.txt
	@cd go/ && ./logcleaner realtest.txt 100 "2025-01-01 00:00:00"

run: build
	@cd go/ && ./logcleaner

# macro
everything: build test run
#	@-git add -A
#	@-git commit -am 'automatic update'
