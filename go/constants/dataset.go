// Copyright 2016 The Noms Authors. All rights reserved.
// Licensed under the Apache License, version 2.0:
// http://www.apache.org/licenses/LICENSE-2.0

package constants

import "regexp"

var DatasetRe = regexp.MustCompile(`[a-zA-Z0-9\-_/]+`)
