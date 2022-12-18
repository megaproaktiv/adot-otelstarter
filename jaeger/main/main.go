package main

import (
	"cluster"
	"github.com/aws/aws-sdk-go-v2/aws"
	"os"

	awscdk "github.com/aws/aws-cdk-go/awscdk/v2"
	"github.com/aws/jsii-runtime-go"
)

func main() {
	defer jsii.Close()

	app := awscdk.NewApp(nil)

	cluster.NewNetworkStack(app, "basevpc",
		&cluster.NetworkStackProps{
			StackProps: awscdk.StackProps{
				Env:         env(),
				Description: aws.String("Base VPC 4 jaeger cluster"),
			},
		},
	)

 	cluster.NewClusterStack(app, "jaeger", &cluster.StackProps{
			StackProps: awscdk.StackProps{
				Env:         env(),
				Description: aws.String("ECS Cluster with jaeger as container"),
			},
		}, 
	)
	

	app.Synth(nil)
}

// env determines the AWS environment (account+region) in which our stack is to
// be deployed. For more information see: https://docs.aws.amazon.com/cdk/latest/guide/environments.html
func env() *awscdk.Environment {

	return &awscdk.Environment{
		Account: jsii.String(os.Getenv("CDK_DEFAULT_ACCOUNT")),
		Region:  jsii.String(os.Getenv("CDK_DEFAULT_REGION")),
	}
}
