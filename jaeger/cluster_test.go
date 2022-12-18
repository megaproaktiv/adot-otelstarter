package cluster_test

import (
	"cluster"
	"os"
	"testing"

	"github.com/aws/aws-cdk-go/awscdk/v2"
	"github.com/aws/aws-cdk-go/awscdk/v2/assertions"
	"github.com/aws/aws-sdk-go-v2/aws"
)

// example tests. To run these tests, uncomment this file along with the
// example resource in cluster_test.go
func TestClusterStack(t *testing.T) {
	// GIVEN
	app := awscdk.NewApp(nil)

	// WHEN
	stack := cluster.NewClusterStack(app, "MyStack", &cluster.StackProps{
		StackProps: awscdk.StackProps{
			Env: env(),
		},
	},)
	// THEN
	template := assertions.Template_FromStack(stack,nil)
	template.HasResourceProperties(aws.String("AWS::ECS::Cluster"), map[string]interface{}{})
}

func env() *awscdk.Environment {

	return &awscdk.Environment{
		Account: aws.String(os.Getenv("CDK_DEFAULT_ACCOUNT")),
		Region:  aws.String(os.Getenv("AWS_REGION")),
	}
}
