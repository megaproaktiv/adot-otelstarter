package otelstarter

import (
	"log"
	"os"
	"path/filepath"

	"github.com/aws/aws-cdk-go/awscdk/v2"
	"github.com/aws/aws-cdk-go/awscdk/v2/awsdynamodb"
	"github.com/aws/aws-cdk-go/awscdk/v2/awsec2"
	"github.com/aws/aws-cdk-go/awscdk/v2/awsiam"
	lambda "github.com/aws/aws-cdk-go/awscdk/v2/awslambda"
	sources "github.com/aws/aws-cdk-go/awscdk/v2/awslambdaeventsources"
	logs "github.com/aws/aws-cdk-go/awscdk/v2/awslogs"
	"github.com/aws/aws-cdk-go/awscdk/v2/awss3"
	"github.com/aws/aws-cdk-go/awscdk/v2/awss3assets"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/constructs-go/constructs/v10"
)

type OtelstarterStackProps struct {
	StackProps awscdk.StackProps
}

func NewOtelstarterStack(scope constructs.Construct, id string, props *OtelstarterStackProps) awscdk.Stack {
	var sprops awscdk.StackProps
	if props != nil {
		sprops = props.StackProps
	}
	stack := awscdk.NewStack(scope, &id, &sprops)

	this := stack
	
	vpc := awsec2.Vpc_FromLookup(this, aws.String("vpc"),
		&awsec2.VpcLookupOptions{
			VpcName: aws.String("otel-base-vpc"),
		},
	)

	path, err := os.Getwd()
	if err != nil {
		log.Println(err)
	}
	lambdaPath := filepath.Join(path, "../dist/main.zip")
	adotLayer := lambda.LayerVersion_FromLayerVersionArn(this, aws.String("adotlayer"), 
		aws.String("arn:aws:lambda:eu-central-1:901920570463:layer:aws-otel-collector-amd64-ver-0-62-1:1"))
	fn := lambda.NewFunction(this, aws.String("adotlambda"), 
	&lambda.FunctionProps{
		Vpc: vpc,
		Description:                  aws.String("otelstarter use otel with lambda go"),
		FunctionName:                 aws.String("otellambda"),
		LogRetention:                 logs.RetentionDays_THREE_MONTHS,
		MemorySize:                   aws.Float64(1024),
		Timeout:                      awscdk.Duration_Seconds(aws.Float64(10)),
		Code: lambda.Code_FromAsset(&lambdaPath, &awss3assets.AssetOptions{}),
		Handler: aws.String("main"),
		Runtime:                      lambda.Runtime_PROVIDED_AL2(),
		Tracing: lambda.Tracing_ACTIVE,
		Environment: &map[string]*string{
			"OPENTELEMETRY_COLLECTOR_CONFIG_FILE" : aws.String("/var/task/config.yml"),
			// "https://opentelemetry.io/docs/concepts/sdk-configuration/general-sdk-configuration/"
			"OTEL_SERVICE_NAME" : aws.String("documentcounter"),
		},
		AllowPublicSubnet: aws.Bool(true),
		Layers: &[]lambda.ILayerVersion{
				adotLayer, 
		},
		},
	)

	
	awscdk.NewCfnOutput(this, aws.String("otel-lambda-out"), &awscdk.CfnOutputProps{
		Value: fn.FunctionName(),
		Description: aws.String("Adot lambda go funktion"),
		ExportName: aws.String("adotlambda-name"),
	})


	bucky := awss3.NewBucket(this, aws.String("incoming-otel"), 
		&awss3.BucketProps{
			BlockPublicAccess: awss3.BlockPublicAccess_BLOCK_ALL(),
		},
	)
	fn.AddEnvironment(aws.String("Bucket"), bucky.BucketName(),nil)

	bucky.GrantRead(fn, "*")
	fn.AddEventSource(sources.NewS3EventSource(bucky, 
		&sources.S3EventSourceProps{
			Events:  &[]awss3.EventType{
					awss3.EventType_OBJECT_CREATED,
			},
			Filters: &[]*awss3.NotificationKeyFilter{},
		}, 
	))
	awscdk.NewCfnOutput(this, aws.String("Bucket"), &awscdk.CfnOutputProps{
		Value:bucky.BucketName(),
		Description: aws.String("Otel lambda go funktion"),
		ExportName: aws.String("BucketName"),
	})
	// *** Event end ****

	// ** Table
	table := awsdynamodb.NewTable(this, aws.String("items"), 
		&awsdynamodb.TableProps{
			PartitionKey:               &awsdynamodb.Attribute{
				Name: aws.String("itemID"),
				Type: awsdynamodb.AttributeType_STRING,
			},
			RemovalPolicy:              awscdk.RemovalPolicy_DESTROY,
			TableName:                  aws.String("items.adot"),
		},
	)
	fn.AddEnvironment(aws.String("TableName"), table.TableName(), nil)
	table.GrantReadWriteData(fn)
	// **

	return stack
}
