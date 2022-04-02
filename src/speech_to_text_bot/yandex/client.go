package yandex

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
)

const (
	storageEndpoint = "https://storage.yandexcloud.net"
)

type YClient struct {
	s3   *s3.S3
	conf *ClientConfig
}

type ClientConfig struct {
	AwsRegion          string
	AwsStaticKeyID     string
	AwsStaticKeySecret string
	AwsApiKey          string
	AwsBucket          string
}

func New(conf *ClientConfig) (*YClient, error) {
	sess, err := connectAws(conf)
	if err != nil {
		return nil, err
	}
	return &YClient{
		s3:   s3.New(sess),
		conf: conf,
	}, nil
}

func connectAws(conf *ClientConfig) (*session.Session, error) {
	return session.NewSession(
		&aws.Config{
			Region:      aws.String(conf.AwsRegion),
			Credentials: credentials.NewStaticCredentials(conf.AwsStaticKeyID, conf.AwsStaticKeySecret, ""),
			Endpoint:    aws.String(storageEndpoint),
		})
}
