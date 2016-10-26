// Copyright 2016 Attic Labs, Inc. All rights reserved.
// Licensed under the Apache License, version 2.0:
// http://www.apache.org/licenses/LICENSE-2.0

package datas

import (
	"github.com/attic-labs/noms/go/merge"
	"github.com/attic-labs/noms/go/types"
)

// CommitOptions is used to pass options into Commit.
type CommitOptions struct {
	// Parents, if provided is the parent commits of the commit we are creating.
	Parents types.Set

	// Meta is a Struct that describes arbitrary metadata about this Commit, e.g. a timestamp or descriptive text.
	Meta types.Struct

	// MergePolicy will be called to attempt to resolve conflicts when merging this Commit. If MergePolicy is nil, no merging will be attempted.
	MergePolicy merge.ResolveFunc
}
