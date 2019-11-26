package xterrafile

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"log"
	"net/url"
	"os"
	"github.com/Azure/azure-storage-blob-go/azblob"
	"path/filepath"
	"strings"
)

func IsAzBlobAStorageUrl(addr string) bool {
	return strings.Contains(addr, "blob.core.windows.net") && strings.HasSuffix(addr, ".tgz")
}

func CopyBlobContent(name string, source string, version string, directory string) {
	DownloadBlob(source, directory)
}

func DownloadBlob(source string, targetDir string) {

	// TODO: get access key

	// https://terraformmodulesst.blob.core.windows.net/cip-gitlab-runner-iaclib/cip-gitlab-runner-iaclib_0.0.62.tgz
	accountName, accountKey := os.Getenv("AZURE_STORAGE_ACCOUNT"), os.Getenv("AZURE_STORAGE_ACCESS_KEY")
	if len(accountName) == 0 || len(accountKey) == 0 {
		log.Fatal("Either the AZURE_STORAGE_ACCOUNT or AZURE_STORAGE_ACCESS_KEY environment variable is not set")
	}

	credential, err := azblob.NewSharedKeyCredential(accountName, accountKey)
	if err != nil {
		log.Fatal("Invalid credentials with error: " + err.Error())
	}
	p := azblob.NewPipeline(credential, azblob.PipelineOptions{})

	ctx := context.Background()
	URL, _ := url.Parse(source)
	blobURL := azblob.NewBlockBlobURL(*URL, p)

	downloadResponse, err := blobURL.Download(ctx, 0, azblob.CountToEnd, azblob.BlobAccessConditions{}, false)

	reader := downloadResponse.Body(azblob.RetryReaderOptions{})
	r, err := gzip.NewReader(reader)
	tarReader := tar.NewReader(r)
	untar(tarReader, targetDir)
	fmt.Println(downloadResponse)
}

func untar(tarReader *tar.Reader, target string) error {
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		} else if err != nil {
			return err
		}

		path := filepath.Join(target, header.Name)
		info := header.FileInfo()
		if info.IsDir() {
			if err = os.MkdirAll(path, info.Mode()); err != nil {
				return err
			}
			continue
		}

		file, err := os.OpenFile(path, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, info.Mode())
		if err != nil {
			return err
		}
		defer file.Close()
		_, err = io.Copy(file, tarReader)
		if err != nil {
			return err
		}
	}
	return nil
}