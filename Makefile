EXTLIBS := github.com/aws/aws-sdk-go/aws         \
           github.com/aws/aws-sdk-go/aws/awserr  \
           github.com/aws/aws-sdk-go/aws/session \
           github.com/aws/aws-sdk-go/aws/request \
           github.com/aws/aws-sdk-go/service/ec2

EXTLIBS_PATH := $(patsubst %, src/%, $(EXTLIBS))

ALL_SOURCES := $(wildcard src/main/*.go)

EXE_SOURCES := $(filter-out %_test.go, $(ALL_SOURCES))


default: all

all: ec2tools

check: ec2tools
	./runtest.sh

ec2tools: $(EXE_SOURCES)
	GOPATH=$(PWD) go build -v -o $@ $(filter src/main/%.go, $^)

test: $(ALL_SOURCES)
	GOPATH=$(PWD) go test -v -timeout 10s $(filter src/main/%.go, $^)


ec2tools test: $(EXTLIBS_PATH)

$(EXTLIBS_PATH): %:
	GOPATH=$(PWD) go get -v $(patsubst src/%, %, $@)


clean:
	-rm -rf ec2tools .ec2tools

cleanall: clean
	-rm -rf src/github.com pkg
