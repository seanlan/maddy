package fs

import (
	"os"
	"testing"

	"mailcoin/framework/module"
	"mailcoin/internal/storage/blob"
	"mailcoin/internal/testutils"
)

func TestFS(t *testing.T) {
	blob.TestStore(t, func() module.BlobStore {
		dir := testutils.Dir(t)
		return &FSStore{instName: "test", root: dir}
	}, func(store module.BlobStore) {
		os.RemoveAll(store.(*FSStore).root)
	})
}
