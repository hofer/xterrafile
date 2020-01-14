package xterrafile

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"fmt"
	"github.com/Azure/azure-sdk-for-go/services/storage/mgmt/2017-06-01/storage"
	"github.com/Azure/azure-storage-blob-go/azblob"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure/auth"
	"io"
	"log"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

func IsAzBlobAStorageUrl(addr string) bool {
	return strings.Contains(addr, "blob.core.windows.net")
}

func CopyBlobContent(name string, source string, version string, directory string) {
	DownloadBlob(name, source, version, directory)
}

func DownloadBlob(name string, source string, version string, targetDir string) {
	downloadUrl := source + "/" + name + "_" + version + ".tgz"
	p := azblob.NewPipeline(loadCredentials(), azblob.PipelineOptions{})

	ctx := context.Background()
	URL, _ := url.Parse(downloadUrl)
	blobURL := azblob.NewBlockBlobURL(*URL, p)

	fmt.Println("Downloading: " + downloadUrl)
	downloadResponse, err := blobURL.Download(ctx, 0, azblob.CountToEnd, azblob.BlobAccessConditions{}, false)
	if err != nil {
		log.Fatal(err.Error())
	}

	reader := downloadResponse.Body(azblob.RetryReaderOptions{})
	r, _ := gzip.NewReader(reader)
	tarReader := tar.NewReader(r)

	parts := strings.Split(targetDir, string(os.PathSeparator))
	newTargetDir := strings.Join(parts[:len(parts)-1],string(os.PathSeparator))

	untar(tarReader, newTargetDir)
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

var sharedCredential *azblob.SharedKeyCredential

func loadCredentials() *azblob.SharedKeyCredential {
	if sharedCredential == nil {
		subscriptionId     := os.Getenv("TERRAFORM_MODULES_STORAGE_ACCOUNT_SUBSCRIPTION_ID")
		resourceGroupName  := os.Getenv("TERRAFORM_MODULES_STORAGE_ACCOUNT_RESOURCE_GROUP")
		storageAccountName := os.Getenv("TERRAFORM_MODULES_STORAGE_ACCOUNT_NAME")
		useMsi             := GetBool(os.Getenv("ARM_USE_MSI"))

		storageAccountsClient := getStorageAccountsClient(useMsi, subscriptionId)

		keys, err := storageAccountsClient.ListKeys(context.Background(), resourceGroupName, storageAccountName)
		if err != nil || len(*keys.Keys) ==0 {
			log.Fatal("Unable to read storage account access keys. Error: " + err.Error())
		}

		key := (*keys.Keys)[0]
		credential, err := azblob.NewSharedKeyCredential(storageAccountName, *key.Value)
		if err != nil {
			log.Fatal("Invalid credentials with error: " + err.Error())
		}
		sharedCredential = credential
	}
	return sharedCredential
}

func getCliStorageAccountsClient(subscriptionId string)  autorest.Authorizer {
	authorizer, err := auth.NewAuthorizerFromCLI()
	if err != nil {
		log.Fatal("No authorization via CLI: " + err.Error())
	}
	return authorizer
}

func getMsiAuthorizer(subscriptionId string)  autorest.Authorizer {
	authorizer, err := auth.NewMSIConfig().Authorizer()
	if err != nil {
		log.Fatal("No authorization via MSI: " + err.Error())
	}
	return authorizer
}

func getStorageAccountsClient(useMsi bool, subscriptionId string) storage.AccountsClient {
	var authorizer autorest.Authorizer
	if useMsi == true {
		authorizer = getMsiAuthorizer(subscriptionId)
	} else {
		authorizer = getCliStorageAccountsClient(subscriptionId)
	}

	storageAccountsClient := storage.NewAccountsClient(subscriptionId)
	storageAccountsClient.Authorizer = authorizer
	storageAccountsClient.AddToUserAgent("xterrafile")
	return storageAccountsClient
}

func GetBool(value string) bool {
	i, err := strconv.ParseBool(value)
	if nil != err {
		return false
	}
	return i
}