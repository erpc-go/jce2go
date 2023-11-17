help : jce2go
	@echo "make build"
	@echo "make test"

build: 
	go build main.go
	mv main jce2go

update:
	go get -u
	go get -u github.com/erpc-go/jce-codec
	go mod tidy

clean :
	rm -rf jce2go

.PHONY : clean jce2go
