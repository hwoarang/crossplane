/*
Copyright 2022 The Crossplane Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Package run implements a convenience CLI to run and test Composition Functions.
package run

import (
	"context"
	"os"
	"time"

	"google.golang.org/protobuf/types/known/durationpb"

	"github.com/crossplane/crossplane-runtime/pkg/errors"

	"github.com/crossplane/crossplane/apis/apiextensions/fn/proto/v1alpha1"
	"github.com/crossplane/crossplane/internal/xfn"
)

// Error strings
const (
	errWriteFIO    = "cannot write FunctionIO YAML to stdout"
	errRunFunction = "cannot run function"
)

// Command runs a Composition function.
type Command struct {
	CacheDir   string        `short:"c" help:"Directory used for caching function images and containers." default:"/xfn"`
	Timeout    time.Duration `help:"Maximum time for which the function may run before being killed." default:"30s"`
	MapRootUID int           `help:"UID that will map to 0 in the function's user namespace. The following 65336 UIDs must be available. Ignored if xfn does not have CAP_SETUID and CAP_SETGID." default:"100000"`
	MapRootGID int           `help:"GID that will map to 0 in the function's user namespace. The following 65336 GIDs must be available. Ignored if xfn does not have CAP_SETUID and CAP_SETGID." default:"100000"`

	Image      string `arg:"" help:"OCI image to run."`
	FunctionIO []byte `arg:"" help:"YAML encoded FunctionIO to pass to the function." type:"filecontent"`
}

// Run a Composition container function.
func (c *Command) Run() error {
	// If we don't have CAP_SETUID or CAP_SETGID, we'll only be able to map our
	// own UID and GID to root inside the user namespace.
	rootUID := os.Getuid()
	rootGID := os.Getgid()
	setuid := xfn.HasCapSetUID() && xfn.HasCapSetGID() // We're using 'setuid' as shorthand for both here.
	if setuid {
		rootUID = c.MapRootUID
		rootGID = c.MapRootGID
	}

	f := xfn.NewContainerRunner(xfn.SetUID(setuid), xfn.MapToRoot(rootUID, rootGID), xfn.WithCacheDir(c.CacheDir))
	rsp, err := f.RunFunction(context.Background(), &v1alpha1.RunFunctionRequest{
		Image: c.Image,
		Input: c.FunctionIO,
		RunFunctionConfig: &v1alpha1.RunFunctionConfig{
			Timeout: durationpb.New(c.Timeout),
		},
	})
	if err != nil {
		return errors.Wrap(err, errRunFunction)
	}

	_, err = os.Stdout.Write(rsp.GetOutput())
	return errors.Wrap(err, errWriteFIO)
}
