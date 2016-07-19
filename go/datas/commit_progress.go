// Copyright 2016 Attic Labs, Inc. All rights reserved.
// Licensed under the Apache License, version 2.0:
// http://www.apache.org/licenses/LICENSE-2.0

package datas

// CommitProgress provides information about the progress of the commit method.
type CommitProgress struct {
	// DoneBytes is the number of bytes that have been sent so far.
	DoneBytes uint64
	// KnownBytes is the total number of bytes that commit needs to send.
	KnownBytes uint64
}
