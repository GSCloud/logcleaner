#@author Fred Brooker <git@gscloud.cz>

all:
	@echo "build | run | test | everything";

build:
	@echo "Building app ...\n"
	@cd go/ && go build -ldflags="-s -w" -o logcleaner .
	@echo "Done."

test: build
	@echo "Testing app ...\n"
	@cd go/ && go test -v .

realtest: build
	@echo "Testing app on pseudo-real data ...\n"
	@cp go/test_log.txt go/realtest.txt
	@cd go/ && ./logcleaner realtest.txt --lines 3000 --date "2025-06-15" --format "2006-01-02"

run: build
	@echo "\n"
	@cd go/ && ./logcleaner --help

# macro
everything: build test realtest run
	@-git add -A
	@-git commit -am 'automatic update'
