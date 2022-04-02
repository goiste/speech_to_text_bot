package yandex

import (
	"bytes"
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/service/s3"
)

func (c *YClient) UploadFile(key string, data []byte) error {
	ctx, cancelFn := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancelFn()

	_, err := c.s3.PutObjectWithContext(ctx, &s3.PutObjectInput{
		Bucket: aws.String(c.conf.AwsBucket),
		Key:    aws.String(key),
		Body:   bytes.NewReader(data),
	})
	if err != nil {
		if aErr, ok := err.(awserr.Error); ok && aErr.Code() == request.CanceledErrorCode {
			err = fmt.Errorf("upload canceled due to timeout: %w", err)
		} else {
			err = fmt.Errorf("failed to upload object: %w", err)
		}
	}

	return err
}

func (c *YClient) DeleteFile(key string) error {
	ctx, cancelFn := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancelFn()

	_, err := c.s3.DeleteObjectWithContext(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(c.conf.AwsBucket),
		Key:    aws.String(key),
	})
	if err != nil {
		if aErr, ok := err.(awserr.Error); ok && aErr.Code() == request.CanceledErrorCode {
			err = fmt.Errorf("deleting canceled due to timeout: %w", err)
		} else {
			err = fmt.Errorf("failed to delete object: %w", err)
		}
	}

	return err
}
