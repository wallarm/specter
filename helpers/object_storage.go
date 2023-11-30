package helpers

import (
	"bytes"
	"context"
	"fmt"
	"github.com/joho/godotenv"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/sirupsen/logrus"
	"io"
	"mime"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type Client struct {
	Minio *minio.Client
}

type MinioConfig struct {
	Endpoint  string `env:"S3_ENDPOINT"`
	AccessKey string `env:"S3_ACCESS_KEY"`
	SecretKey string `env:"S3_SECRET_KEY"`
	UseSSL    bool   `env:"S3_USE_SSL"`
}

func GetConf() (*MinioConfig, error) {

	var err error
	// Load values from .env file
	_ = godotenv.Load()

	endpoint := os.Getenv("S3_ENDPOINT")
	accessKey := os.Getenv("S3_ACCESS_KEY")
	secretKey := os.Getenv("S3_SECRET_KEY")

	var useSSL bool
	useSSLStr := os.Getenv("S3_USE_SSL")
	if useSSLStr != "" {
		useSSL, err = strconv.ParseBool(useSSLStr)
		if err != nil {
			return nil, fmt.Errorf("error parsing S3_USE_SSL: %s", err)
		}
	}

	// Check that all necessary configurations are set
	if endpoint == "" || accessKey == "" || secretKey == "" {
		return nil, fmt.Errorf("not all necessary environment variables are set")
	}

	conf := MinioConfig{
		Endpoint:  endpoint,
		AccessKey: accessKey,
		SecretKey: secretKey,
		UseSSL:    useSSL,
	}

	return &conf, nil
}

func NewClient(endpoint, accessKeyID, secretAccessKey string, useSSL bool) (*Client, error) {
	client, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKeyID, secretAccessKey, ""),
		Secure: useSSL,
	})
	if err != nil {
		logrus.Fatalf("Error create S3 client: %v; endpoint: %v; accessKeyID: %v; "+
			"secretAccessKey: %v; useSSL: %v", err, endpoint, accessKeyID, secretAccessKey, useSSL)
		return nil, err
	}

	//buckets, err := client.ListBuckets(context.Background())
	//if err != nil {
	//	logrus.Errorf("Error list buckets: %v", err)
	//	return nil, err
	//}
	//
	//logrus.Info("======= Buckets name =======")
	//
	//for _, bucket := range buckets {
	//	if strings.HasPrefix(bucket.Name, "wallarm-perf") {
	//		logrus.Infof("Bucket: %v", bucket.Name)
	//	}
	//}

	return &Client{client}, nil
}

type Minio interface {
	Upload(ctx context.Context, bucketName, objectName string, fileContent io.Reader, fileSize int64) error
	GetLink(bucketName, objectName string) (string, error)
	//IsExist(bucketName, objectName string) bool
	UpdateFile(ctx context.Context, bucketName, objectName string) error
}

func (client *Client) Upload(ctx context.Context, bucketName, objectName string, fileContent io.Reader, fileSize int64) error {

	ext := filepath.Ext(objectName)
	contentType := mime.TypeByExtension(ext)

	_, err := client.Minio.PutObject(
		ctx, bucketName, objectName, fileContent, fileSize, minio.PutObjectOptions{
			ContentType: contentType,
			UserMetadata: map[string]string{
				"x-amz-acl": "public-read",
			},
		})

	return err

}

func (client *Client) GetLink(ctx context.Context, bucketName, objectName string) (string, error) {
	reqParams := make(url.Values)

	// ToDo заменить your-filename.txt на имя файла
	//reqParams.Set(constant.ResponseContentDisposition, "inline; filename=\"your-filename.txt\"")
	//reqParams.Set(constant.ResponseContentType, ContentType)

	object, err := client.Minio.PresignedGetObject(
		ctx, bucketName, objectName, time.Second*7*60*60, reqParams)
	if err != nil {
		return "", err
	}
	return object.String(), nil
}

func Initialize() *Client {

	var (
		config              *MinioConfig
		objectStorageClient *Client
		err                 error
	)

	config, err = GetConf()
	if err != nil {
		logrus.Fatalf("Error parsing config: %s", err)
	}

	objectStorageClient, err = NewClient(
		config.Endpoint, config.AccessKey, config.SecretKey, config.UseSSL)
	if err != nil {
		logrus.Fatalf("Error creating S3 client: %s", err)
	}

	return objectStorageClient
}

func UploadFileToS3(ctx context.Context, client *Client, fileName, bucket string) error {
	fileContent, err := os.ReadFile(fileName)
	if err != nil {
		return fmt.Errorf("error reading %s: %s", fileName, err)
	}

	reader := bytes.NewReader(fileContent)
	fileSize := int64(len(fileContent))

	_ = godotenv.Load()
	pipelineID := os.Getenv("CI_PIPELINE_ID")
	if pipelineID == "" {
		return fmt.Errorf("CI_PIPELINE_ID is not set in the environment")
	}

	var newFileName string
	switch {
	case strings.Contains(fileName, "load.yaml"):
		newFileName = fmt.Sprintf("%s/load.yaml", pipelineID)
	case strings.Contains(fileName, "ammo.json"):
		newFileName = fmt.Sprintf("%s/ammo.json", pipelineID)
	case regexp.MustCompile(`(^|/)http_phout\.log$`).MatchString(fileName):
		newFileName = fmt.Sprintf("%s/http_phout.log", pipelineID)
	case regexp.MustCompile(`(^|/)phout\.log$`).MatchString(fileName):
		newFileName = fmt.Sprintf("%s/phout.log", pipelineID)
	case strings.Contains(fileName, "answ.log"):
		newFileName = fmt.Sprintf("%s/answ.log", pipelineID)
	default:
		newFileName = fmt.Sprintf("%s/%s", pipelineID, fileName)
	}

	fmt.Printf("Uploading %s to %s/%s\n", fileName, bucket, newFileName)

	return client.Upload(ctx, bucket, newFileName, reader, fileSize)
}

func PrintFileLinks(bucket string) {
	_ = godotenv.Load()
	pipelineID := os.Getenv("CI_PIPELINE_ID")

	linkLoadYaml := fmt.Sprintf("https://storage.googleapis.com/%s/%s/%s", bucket, pipelineID, "load.yaml")
	linkAmmoJson := fmt.Sprintf("https://storage.googleapis.com/%s/%s/%s", bucket, pipelineID, "ammo.json")

	logrus.Info("======= Upload success =======")
	logrus.Infof(" export S3_LOAD_YAML_URL='%s'", linkLoadYaml)
	logrus.Infof(" export S3_AMMO_JSON='%s'", linkAmmoJson)
}
