package archive

import (
	"archive/tar"
	"compress/gzip"
	"io"
	"testing"

	"github.com/input-output-hk/catalyst-forge/lib/tools/fs/billy"
)

func TestArchiveAndCompress(t *testing.T) {
	fs := billy.NewInMemoryFs()

	err := fs.MkdirAll("testdir/subdir", 0755)
	if err != nil {
		t.Fatal(err)
	}

	fs.WriteFile("testdir/file1.txt", []byte("content of file1"), 0644)
	fs.WriteFile("testdir/subdir/file2.txt", []byte("content of file2"), 0644)

	err = ArchiveAndCompress(fs, "testdir", "output.tar.gz")
	if err != nil {
		t.Fatalf("createTarGz failed: %v", err)
	}

	exists, err := fs.Exists("output.tar.gz")
	if err != nil {
		t.Fatal(err)
	}
	if !exists {
		t.Fatal("output.tar.gz was not created")
	}

	file, err := fs.Open("output.tar.gz")
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()

	gzr, err := gzip.NewReader(file)
	if err != nil {
		t.Fatal(err)
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)

	expectedFiles := map[string]string{
		"file1.txt":        "content of file1",
		"subdir/file2.txt": "content of file2",
	}

	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatal(err)
		}

		if content, ok := expectedFiles[header.Name]; ok {
			buf := make([]byte, header.Size)
			_, err := io.ReadFull(tr, buf)
			if err != nil {
				t.Fatal(err)
			}
			if string(buf) != content {
				t.Errorf("Content mismatch for %s. Expected: %s, Got: %s", header.Name, content, string(buf))
			}
			delete(expectedFiles, header.Name)
		}
	}

	if len(expectedFiles) > 0 {
		t.Errorf("Some expected files were not found in the archive: %v", expectedFiles)
	}
}
