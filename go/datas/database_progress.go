// Copyright 2016 Attic Labs, Inc. All rights reserved.
// Licensed under the Apache License, version 2.0:
// http://www.apache.org/licenses/LICENSE-2.0

package datas

// DatabaseProgress provides information about the progress of the database. This is an
// approximation and it will only give useful information if there is only one client of the
// underlying chunk store client.
type DatabaseProgress struct {
	// DoneBytes is the number of bytes that have been sent so far.
	DoneBytes uint64
	// KnownBytes is the total number of bytes that commit needs to send.
	KnownBytes uint64
}
