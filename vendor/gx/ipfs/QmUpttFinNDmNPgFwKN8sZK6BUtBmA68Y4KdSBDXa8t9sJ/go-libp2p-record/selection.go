package record

import (
	"errors"
)

// A SelectorFunc selects the best value for the given key from
// a slice of possible values and returns the index of the chosen one
type SelectorFunc func(string, [][]byte) (int, error)

type Selector map[string]SelectorFunc

func (s Selector) BestRecord(k string, recs [][]byte) (int, error) {
	if len(recs) == 0 {
		return 0, errors.New("no records given")
	}

	ns, _, err := splitPath(k)
	if err != nil {
		return 0, err
	}

	sel, ok := s[ns]
	if !ok {
		log.Infof("Unrecognized key prefix: %s", ns)
		return 0, ErrInvalidRecordType
	}

	return sel(k, recs)
}

// PublicKeySelector just selects the first entry.
// All valid public key records will be equivalent.
func PublicKeySelector(k string, vals [][]byte) (int, error) {
	return 0, nil
}
