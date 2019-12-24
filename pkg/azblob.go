package xterrafile

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"fmt"
	"github.com/Azure/azure-sdk-for-go/services/storage/mgmt/2017-06-01/storage"
	"github.com/Azure/azure-storage-blob-go/azblob"
	"github.com/Azure/go-autorest/autorest/azure/auth"
	"io"
	"log"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

func IsAzBlobAStorageUrl(addr string) bool {
	return strings.Contains(addr, "blob.core.windows.net")
}

func CopyBlobContent(name string, source string, version string, directory string) {
	DownloadBlob(name, source, version, directory)
}

func DownloadBlob(name string, source string, version string, targetDir string) {

	subscriptionId := os.Getenv("STORAGE_ACCOUNT_SUBSCRIPTION_ID")
	resourceGroupName := os.Getenv("STORAGE_ACCOUNT_RESOURCE_GROUP")
	storageAccountName := os.Getenv("STORAGE_ACCOUNT_NAME")
	var accountKey = ""

	storageAccountsClient := getStorageAccountsClient(subscriptionId)
	keys, err := storageAccountsClient.ListKeys(context.Background(), resourceGroupName, storageAccountName)

	key := (*keys.Keys)[0]
	accountKey = *key.Value
	//
	//for _, key := range *keys.Keys {
	//	fmt.Printf("\tKey name: %s\n\tValue: %s...\n\tPermissions: %s\n",
	//		*key.KeyName,
	//		(*key.Value)[:5],
	//		key.Permissions)
	//	fmt.Println("\t----------------")
	//	accountKey = *key.Value
	//}

	downloadUrl := source + "/" + name + "_" + version + ".tar"

	credential, err := azblob.NewSharedKeyCredential(storageAccountName, accountKey)
	if err != nil {
		log.Fatal("Invalid credentials with error: " + err.Error())
	}
	p := azblob.NewPipeline(credential, azblob.PipelineOptions{})

	ctx := context.Background()
	URL, _ := url.Parse(downloadUrl)
	blobURL := azblob.NewBlockBlobURL(*URL, p)

	downloadResponse, err := blobURL.Download(ctx, 0, azblob.CountToEnd, azblob.BlobAccessConditions{}, false)
	if err != nil {
		log.Fatal(err.Error())
	}

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

func getStorageAccountsClient(subscriptionId string) storage.AccountsClient {
	storageAccountsClient := storage.NewAccountsClient(subscriptionId)
	//authorizer, _ := auth.NewAuthorizerFromFile()
	//authorizer, _ := auth.NewAuthorizerFromEnvironment()
	authorizer, err := auth.NewAuthorizerFromCLI()

	if err != nil {
		log.Fatal("No authorization via CLI: " + err.Error())
	}
	//authorizer.WithAuthorization()
    //auth, _ := iam.GetResourceManagementAuthorizer()
	storageAccountsClient.Authorizer = authorizer
	storageAccountsClient.AddToUserAgent("xterrafile")
	return storageAccountsClient
}