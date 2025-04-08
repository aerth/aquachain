#!/bin/bash
# find_unused_packages.bash - check all go packages and see which we can remove
# part of aquachain (https://aquachain.github.io)

list_all_packages(){
    go list -f '{{.ImportPath}}' ./... | grep -v '/vendor/' | grep -v '/test/' | grep -v '/cmd/' | grep -v '/examples/' | grep -v '/scripts/' | grep -v '/tools/' | grep -v '/third_party/' | grep -v '/testdata/' | grep -v '/pkg/' | grep -v '/internal/'
}

all_pkg=$(list_all_packages)
this_module=$(go list)

check_pkg(){
    pkg=$1
    path=".${pkg#${this_module}}"
    # echo "Checking ${pkg} ($path)"
    if [ ! -d "$path" ]; then
        echo "Directory $path does not exist"
        exit 3
    fi
    for pkg2 in $all_pkg; do
        path2=".${pkg2#${this_module}}"
        # echo path1="$path" path2="$path2"
        if [ "$pkg" != "$pkg2" ]; then
            # echo grep -rn "$pkg" "$path2"
            grep -rn "$pkg" "$path2" 2>/dev/null 1>/dev/null && return 0
        fi
    done
    # echo "No references to $pkg found in other packages"
    return 1
    
}
unused=()
for pkg_ in $all_pkg; do
    check_pkg ${pkg_}
    if [ $? -eq 1 ]; then
        echo "Package $pkg_ is unused"
        path=".${pkg#${this_module}}"
        unused+=("$path")
    fi
done

echo "you can run: rm -rf ${unused[@]}"