package commands

import (
	"fmt"
	"io"
)

type keyValueRow struct {
	Key   string
	Value string
}

func printKeyValueTable(out io.Writer, rows []keyValueRow) error {
	maxKeyLen := 0
	for _, r := range rows {
		if len(r.Key) > maxKeyLen {
			maxKeyLen = len(r.Key)
		}
	}

	for _, r := range rows {
		if _, err := fmt.Fprintf(out, "%-*s  %s\n", maxKeyLen, r.Key, r.Value); err != nil {
			return err
		}
	}
	return nil
}
