# $(info loading release.mk ...)

releasetexts := README.md COPYING AUTHORS

defaultwhat: 
	@echo "release mk file :)"

# TODO remove this line after fixing release directory issue
.PRECIOUS: bin/% tmprelease/bin/%/aquachain tmprelease/bin/%/aquachain.exe tmprelease/bin/%

## package above binaries (eg release/aquachain-0.0.1-windows-amd64/)
.PHONY: debs package package-win deb
package-win: release/$(maincmd_name)-windows-amd64.zip
package: release/$(maincmd_name)-windows-amd64.zip \
	release/$(maincmd_name)-osx-amd64.zip \
	release/$(maincmd_name)-linux-amd64.tar.gz \
	release/$(maincmd_name)-linux-riscv64.tar.gz \
	release/$(maincmd_name)-linux-arm.tar.gz \
	release/$(maincmd_name)-freebsd-amd64.tar.gz \
	release/$(maincmd_name)-openbsd-amd64.tar.gz \
	release/$(maincmd_name)-netbsd-amd64.tar.gz \
	debs

# create debian packages (3 arch)
debs: tmprelease/bin/linux-amd64/aquachain tmprelease/bin/linux-arm/aquachain tmprelease/bin/linux-riscv64/aquachain
	bash contrib/makedeb.bash -d tmprelease/bin linux-amd64 linux-arm linux-riscv64
	mv *.deb release/

# for not cross-compile
aquachain_$(version)_$(GOOS)_$(GOARCH).deb:
	bash contrib/makedeb.bash $(GOOS)-$(GOARCH)

# # create release packages
release/$(maincmd_name)-windows-%.zip: tmprelease/bin/windows-%/
	zip -vr $@ $^/aquachain* ${releasetexts}
	
release/$(maincmd_name)-osx-%.zip: tmprelease/bin/osx-%/
	zip -vr $@ $^/aquachain* ${releasetexts}

# create release binaries
# eg: windows-amd64/aquachain.exe
tmprelease/bin/%: $(GOFILES)
	$(info starting cross-compile $* -> $@)
# compiles to bin/$GOOS-$GOARCH/aquachain or bin/$GOOS-$GOARCH/aquachain.exe
	env GOOS=$(shell echo $* | cut -d- -f1) GOARCH=$(shell echo $* | cut -d- -f2) \
		${MAKE} cross
	mkdir -p $@
	mv -v bin/$*/aquachain* $@
	echo "built $* -> $@"
	file $@/aquachain* || true
tmprelease/bin/%/aquachain.exe: tmprelease/bin/%/aquachain

release/$(maincmd_name)-%.tar.gz: tmprelease/bin/%
	mkdir -p release
	rm -rf tmprelease/${maincmd_name}-$*
	mkdir -p tmprelease/${maincmd_name}-$*
	cp -t tmprelease/${maincmd_name}-$*/aquachain* ${releasetexts}
	cd tmprelease && tar czf ../$@ ${maincmd_name}-$*

# bin/windows-%: bin/windows-%.exe
tmprelease/bin/windows-%/aquachain.exe: 
	$(info building windows-$* -> $@)
	env GOOS=windows GOARCH=$(shell echo $* | cut -d- -f2) \
		${MAKE} cross
	echo "built $* -> $@"
	file $@
tmprelease/bin/%/aquachain:
	$(info building $* -> $@)
	env GOOS=$(shell echo $* | cut -d- -f1) GOARCH=$(shell echo $* | cut -d- -f2) \
		${MAKE} cross
	echo "built $* -> $@"
	file $@

release/$(maincmd_name)-windows-amd64.zip: tmprelease/bin/windows-amd64
	mkdir -p release
	rm -rf tmprelease/${maincmd_name}-windows-amd64
	mkdir -p tmprelease/${maincmd_name}-windows-amd64
	cp -t tmprelease/${maincmd_name}-windows-amd64 $^/aquachain* ${releasetexts}
	cd tmprelease && zip -r ../$@ ${maincmd_name}-windows-amd64
release/$(maincmd_name)-osx-amd64.zip: tmprelease/bin/darwin-amd64
	mkdir -p release
	rm -rf tmprelease/${maincmd_name}-osx-amd64
	mkdir -p tmprelease/${maincmd_name}-osx-amd64
	cp -t tmprelease/${maincmd_name}-osx-amd64 $^/aquachain* ${releasetexts}
	cd tmprelease && zip -r ../$@ ${maincmd_name}-osx-amd64
release/$(maincmd_name)-%.tar.gz: tmprelease/bin/%
	mkdir -p release
	rm -rf tmprelease/${maincmd_name}-$*
	mkdir -p tmprelease/${maincmd_name}-$*
	cp -t tmprelease/${maincmd_name}-$* $^/aquachain* ${releasetexts}
	cd tmprelease && tar czf ../$@ ${maincmd_name}-$*
