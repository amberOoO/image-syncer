package tools

import (
	"encoding/base64"
	"strings"

	"context"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/ecr"
)

type AwsHelper struct {
	client *ecr.Client
}

func NewAwsHelperFromDefaultConfig() *AwsHelper {
	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		panic(err)
	}
	client := ecr.NewFromConfig(cfg)
	return &AwsHelper{client: client}
}

func NewAwsHelperFromStaticConfig(accessKeyID string, accessKey string) (*AwsHelper, error) {
	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(accessKeyID, accessKey, "")),
	)
	if err != nil {
		return nil, err
	}
	// acquire client from static config
	client := ecr.NewFromConfig(cfg)
	// acquire token
	return &AwsHelper{client: client}, err
}

func (ah *AwsHelper) GetAuth() (username string, password string, err error) {
	authOutput, err := ah.client.GetAuthorizationToken(context.TODO(), &ecr.GetAuthorizationTokenInput{})
	if err != nil {
		return "", "", err
	}
	decodedTokenOutput, err := base64.StdEncoding.DecodeString(*authOutput.AuthorizationData[0].AuthorizationToken)
	if err != nil {
		return "", "", err
	}
	username = strings.Split(string(decodedTokenOutput), ":")[0]
	password = strings.Split(string(decodedTokenOutput), ":")[1]
	return username, password, err
}

func (ah *AwsHelper) CreateRepository(repoName string) (*ecr.CreateRepositoryOutput, error) {
	output, err := ah.client.CreateRepository(context.TODO(), &ecr.CreateRepositoryInput{
		RepositoryName: &repoName,
	})
	if err != nil {
		return nil, err
	}
	return output, err
}

func (ah *AwsHelper) IsRepositoryExist(repoName string) bool {
	_, err := ah.client.DescribeRepositories(context.TODO(), &ecr.DescribeRepositoriesInput{
		RepositoryNames: []string{repoName},
	})
	return err == nil
}
