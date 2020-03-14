// +build tools

package restack

import (
	_ "github.com/golang/mock/mockgen"
	_ "github.com/tcnksm/ghr"
	_ "golang.org/x/lint/golint"
	_ "honnef.co/go/tools/cmd/staticcheck"
)
