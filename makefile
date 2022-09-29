run : jce2go
	./jce2go -o demo2go -mod github.com/erpc-go/jce-codec  demo/*

debug : jce2go
	./jce2go -o demo2go -mod github.com/erpc-go/jce-codec  -debug demo/test.jce

jce2go : generate.go lex.go main.go parse.go version.go
	go build .

build : generate.go lex.go main.go parse.go version.go
	go build .

update :
	go get -u
	go get -u github.com/erpc-go/jce-codec@master	
	go mod tidy

help : jce2go
	./jce2go -h

h : jce2go
	./jce2go -h

test:
	go test -v -test.run  TestRequestPacket

clean :
	rm -rf demo2go demo.go

.PHONY : clean jce2go
