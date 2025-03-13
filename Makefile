# edit mkconfig.mk if necessary
include mkconfig.mk
ifeq (,$(GOCMD))
$(warning "warning: go command not found in PATH")
GOCMD=echo
endif
gobindatacmd ?= $(shell which go-bindata)
# for install target
build_dir ?= bin
PREFIX ?= /usr/local
INSTALL_DIR ?= $(PREFIX)/bin

# used for cross-compilation
maybeext := 
ifeq (windows,$(GOOS))
maybeext := .exe
endif

PWD ?= $(shell pwd)


# the main target is bin/aquachain or bin/aquachain.exe
shorttarget := bin/aquachain$(maybeext)
# $(info shorttarget = $(shorttarget))

define LOGO
                              _           _
  __ _  __ _ _   _  __ _  ___| |__   __ _(_)_ __
 / _ '|/ _' | | | |/ _' |/ __| '_ \ / _' | | '_ \ 
| (_| | (_| | |_| | (_| | (__| | | | (_| | | | | |
 \__,_|\__, |\__,_|\__,_|\___|_| |_|\__,_|_|_| |_|
          |_|
	Latest Source: https://gitlab.com/aquachain/aquachain
	Website: https://aquachain.github.io

Target architecture: $(GOOS)/$(GOARCH)
Version: $(version) (commit=$(COMMITHASH)) $(codename)
endef

# apt install file 

# targets
$(shorttarget):
# if on windows, this would become .exe.exe but whatever
$(shorttarget).exe: $(GOFILES)
	$(info Building... $@)
	GOOS=windows CGO_ENABLED=$(CGO_ENABLED) $(GOCMD) build -tags '$(tags)' $(GO_FLAGS) -o $@ $(aquachain_cmd)
	@echo compiled: $(shorttarget)
	@sha256sum $(shorttarget) 2>/dev/null || true
	@file $(shorttarget) 2>/dev/null || true
.PHONY += install
install:
	install -v bin/aquachain $(INSTALL_DIR)/
.PHONY += install commandlist default print-version
default: $(shorttarget)
version: print-version
print-version:
	@echo $(version)
echoflags:
	@echo "CGO_ENABLED=$(CGO_ENABLED) $(GOCMD) build -tags '$(tags)' $(GO_FLAGS) -o $@ $(aquachain_cmd)"
echo:
	$(info  )
	$(info Variables:)
	$(info  )
	@$(foreach V,$(.VARIABLES), \
		$(if $(filter-out environment% default automatic, $(origin $V)), \
			$(if $(filter-out LOGO GOFILES,$V), \
				$(info $V=$($V)) )))
	$(info  )
clean:
	rm -rf bin release docs tmprelease tmpaqua-short-tests.*.tmp
bootnode: bin/aquabootnode
run/%${maybeext}: bin/%${maybeext}
	$(info Running... $<)
ifeq (,$(args))
	$(info No arguments provided, use something like make run args='-h' $@ to add args)
endif
ifeq ($(GOOS),windows)
	$< $(args)
else
	./$< ${args}
endif
bin/%${maybeext}: $(GOFILES)
	$(info Building command ... ./cmd/$*)
	$(info $(LOGO))
	$(info Building... $@)
	CGO_ENABLED=$(CGO_ENABLED) $(GOCMD) build -tags '$(tags)' $(GO_FLAGS) -o $@ ./cmd/$*
	@echo Compiled: $@
	@sha256sum $@ 2>/dev/null || echo "warn: 'sha256sum' command not found"
	@file $@ 2>/dev/null || echo "warn: 'file' command not found"
	@ls -lh $@ 2>/dev/null
# helper target to list all available bin/ commands
# TODO: remove internal and utils from this list
commandlist:
	@echo "Available commands to compile:"
	@ls -1 cmd/ | egrep -v '^_|^internal|^utils' | sed -E 's/^(.*)$$/    make bin\/\1${maybeext}/'
.PHONY += default bootnode hash
deb: aquachain_$(version)_$(GOOS)_$(GOARCH).deb
internal/jsre/deps/bindata.go: internal/jsre/deps/web3.js  internal/jsre/deps/bignumber.js
	@test -x "$(gobindatacmd)" || echo 'warn: go-bindata not found in PATH. run make devtools to install required development dependencies'
	@test -x "$(gobindatacmd)" || exit 0
	@echo "regenerating embedded javascript assets"
# TODO: why is this bit needed now? maybe go-bindata version doesnt touch file
	@rm -vf internal/jsre/deps/bindata.go
	@test ! -x "$(gobindatacmd)" || go generate -v -x ./internal/jsre/deps/...
regen:
	@echo "regenerating embedded assets"
	@test -x "$(gobindatacmd)" || echo 'warn: go-bindata not found in PATH. run make devtools to install required development dependencies'
	@test ! -x "$(gobindatacmd)" || go generate -x ./...
all:
	mkdir -p bin
	cd bin && \
		CGO_ENABLED=$(CGO_ENABLED) ${GOCMD} build -o . -tags '$(tags)' $(GO_FLAGS) ../cmd/...

main_command_dir := ${aquachain_cmd}

# cross compilation wizard target for testing different architectures
cross:
	@test -n "$(GOOS)"
	@test -n "$(GOARCH)" 
	test "GOOS=$(GOOS)" != "GOOS=aquachain"
	@echo cross-compiling for $(GOOS)/$(GOARCH)
ifneq (1,$(release))
	$(info warn: to build a real release, use "make clean release release=1")
else
	$(info Building release version for $(GOOS)/$(GOARCH) (release=1))
endif
	@mkdir -p bin/${GOOS}-${GOARCH}
	$(info Building to directory: bin/${GOOS}-${GOARCH})
	cd bin/${GOOS}-${GOARCH} && GOOS=${GOOS} GOARCH=${GOARCH} \
		CGO_ENABLED=$(CGO_ENABLED) ${GOCMD} build -o . -tags '$(tags)' $(GO_FLAGS) ../.${main_command_dir}
	@echo "compiled: bin/${GOOS}-${GOARCH}/*"
	@sha256sum bin/${GOOS}-${GOARCH}/aquachain$(winextension) 2>/dev/null || true
	@file bin/${GOOS}-${GOARCH}/aquachain$(winextension) 2>/dev/null || true
	@ls -lh bin/${GOOS}-${GOARCH}/aquachain$(winextension) 2>/dev/null || true

help: commandlist
	@echo Variables:
	@echo PREFIX="$(PREFIX)/"
	@echo INSTALL_DIR="$(INSTALL_DIR)/"
	@echo without args, target is: "$(shorttarget)"
	@echo using go flags: "-tags '$(tags)' $(GO_FLAGS)"
	@echo
	@echo "Help:"
	@echo To compile quickly, run \'make\' and then: $(shorttarget) version
	@echo To install system-wide, run something like: sudo make install
	@echo
	@echo -n "note: adding 'release=1' to any target builds a release version\n"
	@echo
	@echo -n "make\n\tdefault, compile bin/aquachain${winextension}\n"
	@echo -n "make cross GOOS=linux GOARCH=amd64\n\tcross-compile for a different OS/arch\n"
	@echo -n "make clean release=1 cross deb\n\tcreate a Debian package (.deb file)\n"
	@echo -n "make clean release=1 release\n\tcreate all release files for all OS/arch combos\n"
	@echo -n "make echo\n\tprint all variables (for development)\n"
	@echo
	@echo -n "Required commands:\n\tgit, go, file\n"
test:
	CGO_ENABLED=0 bash testing/test-short-only.bash $(args)
race:
	CGO_ENABLED=1 bash testing/test-short-only.bash -race

ifeq (1,$(release))
include release.mk
endif

.PHONY += release
checkrelease:
ifneq (1,$(release))
	echo "use make release release=1"
	exit 1
endif
release: checkrelease package hash
release/SHA384.txt:
	$(hashfn) release/*.tar.gz release/*.zip release/*.deb | tee $@
hash: release/SHA384.txt
.PHONY += hash
devtools: # install required go tools to generate code (only when modifying non-go files)
	bash contrib/install_devtools.sh

generate: devtools
	${GOCMD} generate ./...

goget:
	CGO_ENABLED=$(CGO_ENABLED) ${GOCMD} get -v -t -d ./...

linter: bin/golangci-lint
	CGO_ENABLED=0 ./bin/golangci-lint -v run \
	  --config .golangci.yml \
	  --build-tags static,netgo,osusergo \
	  -v

bin/golangci-lint:
	curl -sfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh| sh -s $(golangci_linter_version)

docs: mkdocs.yml Documentation/*/*
	@echo building docs
	mkdocs build

docker:
	docker build -t aquachain/aquachain .

doc-print-env:
	@echo Bool Env:
	@egrep -rn 'EnvBool\("([^"]+)"\)' | sed -e 's/.*EnvBool("\([^"]\+\)").*/\1/' | sort -u
	@echo String Env:
	@egrep -rn 'sense.(LookupEnv|Getenv)\("([^"]+)"\)' | sed -e 's/.*("\([^"]\+\)").*/\1/' | sort -u

rundev: bin/aquachain
	@test "$(PWD)" != "/workspaces/aquachain" || (echo "warn: not in /workspaces/aquachain" && sleep 1)
	@test "$(PWD)" != "/workspaces/aquachain" || (echo "warn: not in /workspaces/aquachain" && sleep 1)
	@test "$(PWD)" != "/workspaces/aquachain" || (echo "warn: not in /workspaces/aquachain" && sleep 1)
	@echo "serving on 0.0.0.0 (8543 and 8544) and allowing "127.0.0.1/24,172.18.0.1/32" for rpc and ws"
	@echo "using ./tmpdatadir as datadir"
	@echo "from host, can connect to the node with: aquachain attach ./tmpdatadir/aquachain.ipc"
	@echo
	@echo "to use arguments, run this from the container:"
	@echo  "    make rundev args='-now help"
	@echo 
	@echo Press ENTER to continue or wait 3 seconds
	@bash -c 'read -t 3 || true'
	./bin/aquachain -rpc -rpcaddr 0.0.0.0 -ws -wsaddr 0.0.0.0 -allowip "127.0.0.1/24,172.18.0.1/32" \
		-datadir tmpdatadir -debug ${args}
