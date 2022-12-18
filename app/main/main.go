package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/smithy-go"

	awshttp "github.com/aws/aws-sdk-go-v2/aws/transport/http"

	"go.opentelemetry.io/contrib/instrumentation/github.com/aws/aws-lambda-go/otellambda"
	"go.opentelemetry.io/contrib/instrumentation/github.com/aws/aws-lambda-go/otellambda/xrayconfig"
	"go.opentelemetry.io/contrib/instrumentation/github.com/aws/aws-sdk-go-v2/otelaws"
	"go.opentelemetry.io/contrib/propagators/aws/xray"
	"go.opentelemetry.io/otel"
)

var ClientDDB *dynamodb.Client
var ClientS3 *s3.Client

// Item holds Dynamodb input
type Item struct {
	ItemID string `json:"itemID"`
    Time  string `json:"time"`
}


// MyEvent Struct for S3 event
type MyEvent struct {
	Name string `json:"name"`
}

func init() {
	// Initialize AWS config.
	cfg, err := awsconfig.LoadDefaultConfig(context.TODO())
	if err != nil {
		panic("configuration error, " + err.Error())
	}
	// Instrument all AWS clients.
	otelaws.AppendMiddlewares(&cfg.APIOptions)
	// Create an instrumented S3 client from the config.
	ClientDDB = dynamodb.NewFromConfig(cfg)
	ClientS3 = s3.NewFromConfig(cfg)
}	
	

func HandleRequest(ctx context.Context, s3Event events.S3Event) (string, error) {
	
		s3input := ExtractKey(s3Event);

		fmt.Printf("Key = %s \n", s3input)

		putItem(ctx,s3input)

		doSomethingWithS3(ctx)

		return os.Getenv("_X_AMZN_TRACE_ID"), nil
	
}

func main() {
	ctx := context.Background()

	tp, err := xrayconfig.NewTracerProvider(ctx)
	if err != nil {
		fmt.Printf("error creating tracer provider: %v", err)
	}

	defer func(ctx context.Context) {
		err := tp.Shutdown(ctx)
		if err != nil {
			fmt.Printf("error shutting down tracer provider: %v", err)
		}
	}(ctx)

	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(xray.Propagator{})

	lambda.Start(otellambda.InstrumentHandler(HandleRequest, xrayconfig.WithRecommendedOptions(tp)... ))
}


func putItem(ctx context.Context, itemID string){

	tableName := os.Getenv("TableName")


	// Create DynamoDB client
	
	t := time.Now()

	input := &dynamodb.PutItemInput{
        Item: map[string]types.AttributeValue{
            "itemID": &types.AttributeValueMemberS{
                Value: itemID,
			},
			"time" : &types.AttributeValueMemberS{
				Value: t.String(),
			},
        },
        TableName: aws.String(tableName),
    }
  
	result, err := ClientDDB.PutItem(ctx,input)
	if err != nil {
		// To get a specific API error
			var notFoundErr *types.ResourceNotFoundException
			if errors.As(err, &notFoundErr) {
				log.Printf("scan failed because the table was not found, %v",
					notFoundErr.ErrorMessage())
			}
	
			// To get any API error
			var apiErr smithy.APIError
			if errors.As(err, &apiErr) {
				log.Printf("scan failed because of an API error, Code: %v, Message: %v",
					apiErr.ErrorCode(), apiErr.ErrorMessage())
			}
	
			// To get the AWS response metadata, such as RequestID
			var respErr *awshttp.ResponseError // Using import alias "awshttp" for package github.com/aws/aws-sdk-go-v2/aws/transport/http
			if errors.As(err, &respErr) {
				log.Printf("scan failed with HTTP status code %v, Request ID %v and error %v",
					respErr.HTTPStatusCode(), respErr.ServiceRequestID(), respErr)
			}
	
			return
			
		}


	fmt.Println("Successfully added " ,result)
}


func ExtractKey(s3event  events.S3Event) string{
	
	return s3event.Records[0].S3.Object.Key;
}

func doSomethingWithS3(ctx context.Context){
	bucketName := os.Getenv("Bucket")

	_, err := ClientS3.ListObjectsV2(ctx, &s3.ListObjectsV2Input{
		Bucket:              &bucketName,
		MaxKeys:             0,
	})
	if err != nil {
		log.Fatal("S3 problem: ",err)
	}
}