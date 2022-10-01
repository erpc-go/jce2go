help : jce2go
	./jce2go -h

run : jce2go
	./jce2go -o demo2go -mod github.com/erpc-go/jce2go  demo/*

debug : jce2go
	./jce2go -o demo2go -mod github.com/erpc-go/jce2go  -debug demo/test.jce

jce2go : generate.go lex.go main.go parse.go version.go
	go build .

build : generate.go lex.go main.go parse.go version.go
	go build .

update :
	go get -u
	go get -u github.com/erpc-go/jce-codec@main
	go mod tidy

test:
	go test -v -test.run  TestRequestPacket

clean :
	rm -rf demo2go/ demo.go jce2go

.PHONY : clean jce2go
