SILKROAD := ./silkroad

GO_FILES := $(shell find . -path './testdata' -prune -o -type f -name '*.go' -print)

$(SILKROAD): $(GO_FILES)
	go build -o $@ -v

.PHONY: update-artifacts
update-artifacts: $(SILKROAD)
	./silkroad -p testdata -o test.dot
	dot -Tsvg test.dot > test.svg
	./silkroad -p testdata -o test2.dot --ignore-external --go-mod-path .
	dot -Tsvg test2.dot > test2.svg
	./silkroad -p testdata --package-pattern ./t1/...,./t2 -o test3.dot
	dot -Tsvg test3.dot > test3.svg
