package archive

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"io"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCreateFromBlobs(t *testing.T) {
	entries := []BlobEntry{
		{Path: "/rill.yaml", Data: []byte("compiler: rillv1\n")},
		{Path: "/models/orders.sql", Data: []byte("SELECT 1")},
		{Path: "/data/blob.bin", Data: []byte{0x00, 0xff, 0x10}},
		{Path: "/.env", Data: []byte("SECRET=1")}, // must be ignored
	}

	buf, err := CreateFromBlobs(context.Background(), entries)
	require.NoError(t, err)

	gz, err := gzip.NewReader(buf)
	require.NoError(t, err)
	tr := tar.NewReader(gz)

	got := map[string][]byte{}
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		require.NoError(t, err)
		data, err := io.ReadAll(tr)
		require.NoError(t, err)
		got[hdr.Name] = data
	}

	require.Len(t, got, 3)
	require.Equal(t, []byte("compiler: rillv1\n"), got["rill.yaml"])
	require.Equal(t, []byte("SELECT 1"), got["models/orders.sql"])
	require.Equal(t, []byte{0x00, 0xff, 0x10}, got["data/blob.bin"])
	require.NotContains(t, got, ".env")
}

func TestCreateFromBlobsRejectsUnsafePaths(t *testing.T) {
	for _, p := range []string{"/../escape.txt", "/a/../../b.txt", "a/./b.txt"} {
		_, err := CreateFromBlobs(context.Background(), []BlobEntry{{Path: p, Data: []byte("x")}})
		require.Error(t, err, "path %q should be rejected", p)
	}
}
