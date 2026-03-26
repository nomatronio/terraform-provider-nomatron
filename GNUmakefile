name = nomatron
organization = nomatronio
version = 1.0.0
arch = darwin_amd64
TFPLUGINDOCS = go run github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs
TFDOCS = go run github.com/terraform-docs/terraform-docs
TFDOCS_DIR ?= .

.PHONY: docs provider-docs tfdocs build install test

docs:
	$(TFPLUGINDOCS) generate --examples-dir=./examples

provider-docs: docs

tfdocs:
	$(TFDOCS) markdown table $(TFDOCS_DIR)

build:
	go build -o bin/terraform-provider-$(name)_v$(version)

install: build
	mkdir -p ~/.terraform.d/plugins/local/$(organization)/$(name)/$(version)/$(arch)
	mv bin/terraform-provider-$(name)_v$(version) ~/.terraform.d/plugins/local/$(organization)/$(name)/$(version)/$(arch)/
test:
	go test ./internal/provider -v
