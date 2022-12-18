module lambda-go

go 1.13

require (
	github.com/aws/aws-lambda-go v1.34.1
	github.com/aws/aws-sdk-go-v2 v1.17.2
	github.com/aws/aws-sdk-go-v2/config v1.17.10
	github.com/aws/aws-sdk-go-v2/service/dynamodb v1.17.3
	github.com/aws/aws-sdk-go-v2/service/s3 v1.29.5
	github.com/aws/smithy-go v1.13.5
	go.opentelemetry.io/contrib/instrumentation/github.com/aws/aws-lambda-go/otellambda v0.36.4
	go.opentelemetry.io/contrib/instrumentation/github.com/aws/aws-lambda-go/otellambda/xrayconfig v0.36.4
	go.opentelemetry.io/contrib/instrumentation/github.com/aws/aws-sdk-go-v2/otelaws v0.36.4
	go.opentelemetry.io/contrib/propagators/aws v1.11.1
	go.opentelemetry.io/otel v1.11.1
	golang.org/x/net v0.0.0-20220225172249-27dd8689420f // indirect
)
