module github.com/AliyunContainerService/image-syncer

go 1.15

require (
	cloud.google.com/go/compute v1.6.1 // indirect
	github.com/aws/aws-sdk-go-v2/config v1.15.11
	github.com/aws/aws-sdk-go-v2/credentials v1.12.6
	github.com/aws/aws-sdk-go-v2/service/ecr v1.17.6
	github.com/containers/image/v5 v5.7.0
	github.com/google/uuid v1.3.0
	github.com/kr/pretty v0.3.0 // indirect
	github.com/prometheus/client_golang v1.11.1 // indirect
	github.com/sirupsen/logrus v1.7.0
	github.com/spf13/cobra v1.1.1
	github.com/stretchr/testify v1.7.1
	github.com/tidwall/gjson v1.9.3
	golang.org/x/net v0.0.0-20220520000938-2e3eb7b945c2 // indirect
	golang.org/x/oauth2 v0.0.0-20220411215720-9780585627b5
	golang.org/x/sys v0.0.0-20220520151302-bc2c85ada10a // indirect
	gopkg.in/yaml.v2 v2.4.0
	gopkg.in/yaml.v3 v3.0.0 // indirect
	gorm.io/driver/sqlite v1.3.4
	gorm.io/gorm v1.23.6
)
