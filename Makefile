#@author Fred Brooker <git@gscloud.cz>

all:
	@echo "build | run | test | everything";

build:
	@echo "Building app ...\n"
	@cd go/ && go build -o logcleaner .
	@echo "Done."

test: build
	@echo "Testing app ...\n"
	@cd go/ && go test -v .

realtest: build
	@echo "Testing app on pseudo-real data ...\n"
	@cp go/test_log.txt go/realtest.txt
	@cd go/ && ./logcleaner realtest.txt 3000 "2025-06-15 00:00:00"

run: build
	@echo "\n"
	@cd go/ && ./logcleaner

# macro
everything: build test realtest run
	@-git add -A
	@-git commit -am 'automatic update'
