GO_MOD_DIRS := $(shell find . -type f -name 'go.mod' -exec dirname {} \; | sort)

.PHONY: unittest
unittest: 
	go test -race -covermode=atomic -v -coverprofile=coverage.txt $(extend) ./...;
	for dir in `find . -type f -name "go.mod" -exec dirname {} \;`; do \
		if [ $${dir} != "." ]; then \
			go test -race -covermode=atomic -v -coverprofile=$${dir}/coverage.txt $(extend) $${dir}/...; \
			if [ -f $${dir}/coverage.txt ]; then \
				tail -n+2 $${lines} $${dir}/coverage.txt >> coverage.txt; \
				rm $${dir}/coverage.txt; \
			fi; \
		fi; \
	done	

.PHONY: benchmark
benchmark: 
	for dir in $(GO_MOD_DIRS); do \
		go test -bench=. -run=^Benchmark $${dir}/...; \
	done

.PHONY: server
server:
	@go run cmd/main.go


.PHONY: mod_tidy
mod_tidy:
	set -e; for dir in $(GO_MOD_DIRS); do \
		echo "go mod tidy in $${dir}"; \
		(cd "$${dir}" && \
		go mod tidy); \
	done