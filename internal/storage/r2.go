package storage

import (
	"bytes"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
)

type R2Client struct {
	client *s3.S3
	bucket string
}

func NewR2Client(endpoint, accessKeyID, secretAccessKey, bucketName string) (*R2Client, error) {
	sess, err := session.NewSession(&aws.Config{
		Region:      aws.String("auto"),
		Endpoint:    aws.String(endpoint),
		Credentials: credentials.NewStaticCredentials(accessKeyID, secretAccessKey, ""),
	})
	if err != nil {
		return nil, err
	}

	return &R2Client{
		client: s3.New(sess),
		bucket: bucketName,
	}, nil
}

func (r *R2Client) UploadGIF(key string, data []byte) error {
	_, err := r.client.PutObject(&s3.PutObjectInput{
		Bucket:      aws.String(r.bucket),
		Key:         aws.String(key),
		Body:        bytes.NewReader(data),
		ContentType: aws.String("image/gif"),
	})
	return err
}

func (r *R2Client) DeleteObject(key string) error {
	_, err := r.client.DeleteObject(&s3.DeleteObjectInput{
		Bucket: aws.String(r.bucket),
		Key:    aws.String(key),
	})
	return err
}
