EXTLIBS := github.com/aws/aws-sdk-go/aws         \
           github.com/aws/aws-sdk-go/aws/awserr  \
           github.com/aws/aws-sdk-go/aws/session \
           github.com/aws/aws-sdk-go/aws/request \
           github.com/aws/aws-sdk-go/service/ec2

EXTLIBS_PATH := $(patsubst %, src/%, $(EXTLIBS))


default: all

all: ec2tools

check: ec2tools
	./ec2tools launch --price=0.035 --region=ap-southeast-2 --size=2 \
                   --key=gauthier sydney
	sleep 30
	./ec2tools get --update instances
	sleep 30
	./ec2tools scp ec2tools
	./ec2tools ssh ls -la
	./ec2tools stop


ec2tools: src/main/context.go src/main/get.go src/main/help.go \
          src/main/launch.go src/main/main.go src/main/scp.go src/main/ssh.go \
          src/main/stop.go src/main/update.go
	GOPATH=$(PWD) go build -v -o $@ $(filter src/main/%.go, $^)

test: src/main/context.go src/main/context_test.go src/main/get.go src/main/help.go \
          src/main/launch.go src/main/main.go src/main/scp.go src/main/ssh.go \
          src/main/stop.go src/main/update.go
	GOPATH=$(PWD) go test $(filter src/main/%.go, $^)


ec2tools test: $(EXTLIBS_PATH)

$(EXTLIBS_PATH): %:
	GOPATH=$(PWD) go get -v $(patsubst src/%, %, $@)


clean:
	-rm -rf ec2tools .ec2tools

cleanall: clean
	-rm -rf src/github.com pkg
