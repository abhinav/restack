#!/bin/bash -e

if [[ -z "$GITHUB_TOKEN" ]]; then
	echo "GITHUB_TOKEN is not set"
	exit 1
fi

VERSION="$1"
if [[ -z "$VERSION" ]]; then
	echo "USAGE: $0 VERSION"
	exit 1
fi

CMDS=(restack)
BUILDS=(
	"darwin amd64"
	"linux amd64"
	"linux arm"
)

build() {
	os="$1"
	arch="$2"
	releasedir="releases/${os}_${arch}"
	tarname="restack.$VERSION.$os.$arch.tar"

	mkdir -p "$releasedir"
	for cmd in "${CMDS[@]}"; do
		GOOS="$os" GOARCH="$arch" go build -o "$releasedir/$cmd" "./cmd/$cmd"
	done

	tar -cf "releases/$tarname" -C "$releasedir" .
	gzip "releases/$tarname"
	rm -r "$releasedir"
}

CHANGELOG=$(go run scripts/extract_changelog.go "$VERSION")

echo "Releasing $VERSION"
echo ""
echo "CHANGELOG:"
echo "$CHANGELOG"
echo ""

for b in "${BUILDS[@]}"; do
	parts=($b)
	os=${parts[0]}
	arch=${parts[1]}
	echo "Building for $os $arch"
	build $os $arch
done

ghr \
	-username abhinav \
	-token "$GITHUB_TOKEN" \
	-body "$CHANGELOG" \
	"$VERSION" releases/
