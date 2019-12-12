build:
	docker run -it --workdir=/go/src/github.com/appspero/aws-identity-provider -v $(PWD):/go/src/github.com/appspero/aws-identity-provider golang:1.12.14 go build .
	zip aws-identity-provider.zip aws-identity-provider

.PHONY : clean
clean:
	-rm -rf aws-identity-provider aws-identity-provider.zip