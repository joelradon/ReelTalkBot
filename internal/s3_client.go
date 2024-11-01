package internal

import (
	"os"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
)

// S3ClientInterface defines methods for S3 operations
type S3ClientInterface interface {
	GetObject(input *s3.GetObjectInput) (*s3.GetObjectOutput, error)
	PutObject(input *s3.PutObjectInput) (*s3.PutObjectOutput, error)
}

// S3Client wraps the AWS S3 client
type S3Client struct {
	Client *s3.S3
}

// NewS3Client initializes a new S3 client with the provided endpoint and region
func NewS3Client(endpoint, region string) S3ClientInterface {
	// Use credentials from environment variables
	awsAccessKeyID := os.Getenv("AWS_ACCESS_KEY_ID")
	awsSecretAccessKey := os.Getenv("AWS_SECRET_ACCESS_KEY")

	creds := credentials.NewStaticCredentials(awsAccessKeyID, awsSecretAccessKey, "")

	// Do not modify the region; use it as provided
	sess := session.Must(session.NewSession(&aws.Config{
		Region:           aws.String(region),
		Endpoint:         aws.String(endpoint),
		Credentials:      creds,
		S3ForcePathStyle: aws.Bool(true), // Needed for custom endpoints
	}))

	return &S3Client{
		Client: s3.New(sess),
	}
}

// GetObject retrieves an object from S3
func (s *S3Client) GetObject(input *s3.GetObjectInput) (*s3.GetObjectOutput, error) {
	return s.Client.GetObject(input)
}

// PutObject uploads an object to S3
func (s *S3Client) PutObject(input *s3.PutObjectInput) (*s3.PutObjectOutput, error) {
	return s.Client.PutObject(input)
}
