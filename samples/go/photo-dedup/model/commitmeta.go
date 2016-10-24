package model

import (
	"time"

	"github.com/attic-labs/noms/go/marshal"
	"github.com/attic-labs/noms/go/types"
	"github.com/attic-labs/noms/go/d"
)

type CommitMeta interface {
	Marshal() types.Struct
}

type commitMeta struct {
	Date string
}

func NewCommitMeta() CommitMeta {
	return &commitMeta{time.Now().Format(time.RFC3339)}
}

func (c *commitMeta) Marshal() types.Struct {
	v, err := marshal.Marshal(*c)
	d.Chk.NoError(err)
	return v.(types.Struct)
}
