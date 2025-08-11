package fs

import (
	"os"
	"testing"

	"github.com/dsoftgames/MailChat/framework/module"
	"github.com/dsoftgames/MailChat/internal/storage/blob"
	"github.com/dsoftgames/MailChat/internal/testutils"
)

func TestFS(t *testing.T) {
	blob.TestStore(t, func() module.BlobStore {
		dir := testutils.Dir(t)
		return &FSStore{instName: "test", root: dir}
	}, func(store module.BlobStore) {
		os.RemoveAll(store.(*FSStore).root)
	})
}
