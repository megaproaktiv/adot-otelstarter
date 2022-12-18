package cluster

import (
	"github.com/aws/aws-cdk-go/awscdk/v2"
	"github.com/aws/aws-cdk-go/awscdk/v2/awsec2"
	"github.com/aws/aws-cdk-go/awscdk/v2/awsssm"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/constructs-go/constructs/v10"
)


type NetworkStackProps struct {
	StackProps awscdk.StackProps
}

func NewNetworkStack(scope constructs.Construct, id string, props *NetworkStackProps) awscdk.Stack{
	var sprops awscdk.StackProps
	if props != nil {
		sprops = props.StackProps
	}
	stack := awscdk.NewStack(scope, &id, &sprops)
	this := stack

	cidr := aws.String("10.0.0.0/20")
	publicSubnet := "Public"
	privateSubnet := "Private"

	vpc := awsec2.NewVpc(this, aws.String("otel-base-vpc"),
		&awsec2.VpcProps{
			VpcName: aws.String("otel-base-vpc"),
			IpAddresses: awsec2.IpAddresses_Cidr(cidr),
			MaxAzs:      aws.Float64(2),
			NatGateways: aws.Float64(1),
			SubnetConfiguration: &[]*awsec2.SubnetConfiguration{
				{
					Name:       &publicSubnet,
					SubnetType: awsec2.SubnetType_PUBLIC,
				},
				{
					Name:       &privateSubnet,
					SubnetType: awsec2.SubnetType_PRIVATE_WITH_EGRESS,

				},
			},
		},
	)

	awsssm.NewStringParameter(stack, aws.String("otel-vpc"),
		&awsssm.StringParameterProps{
		Description:   aws.String("otel base vpc"),
		ParameterName: aws.String("/otel/vpcid"),
		StringValue:   vpc.VpcId(),
	},
)

	return stack
}
