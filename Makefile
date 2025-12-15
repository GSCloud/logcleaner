#@author Fred Brooker <git@gscloud.cz>

all:
	@echo "build | run | test | everything";

build:
	@echo "Building app ..."
	@cd go/ && go build -o logcleaner .

test: build
	@cd go/ && go test -v .

run: build
	@cd go/ && ./logcleaner

# macro
everything: build run
#	@-git add -A
#	@-git commit -am 'automatic update'
