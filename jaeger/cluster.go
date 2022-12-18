package cluster

import (
	"log"

	"github.com/aws/aws-cdk-go/awscdk/v2"
	"github.com/aws/aws-cdk-go/awscdk/v2/awscertificatemanager"
	ec2 "github.com/aws/aws-cdk-go/awscdk/v2/awsec2"
	ecs "github.com/aws/aws-cdk-go/awscdk/v2/awsecs"
	"github.com/aws/aws-cdk-go/awscdk/v2/awsiam"
	"github.com/aws/aws-cdk-go/awscdk/v2/awslogs"
	"github.com/aws/aws-cdk-go/awscdk/v2/awsroute53"
	"github.com/aws/aws-cdk-go/awscdk/v2/awsroute53targets"
	"github.com/aws/aws-cdk-go/awscdk/v2/awsservicediscovery"
	"github.com/aws/aws-cdk-go/awscdk/v2/awsssm"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/constructs-go/constructs/v10"
	
	elbv2 "github.com/aws/aws-cdk-go/awscdk/v2/awselasticloadbalancingv2"
	// Parameter
	paddle "github.com/PaddleHQ/go-aws-ssm"
)

var ConfigurationParams *paddle.Parameters

func init() {
	pmstore, err := paddle.NewParameterStore()
	if err != nil {
		log.Fatal("Cant connect to Parameter Store")
	}
	//Requesting the base path
	ConfigurationParams, err = pmstore.GetAllParametersByPath("/otel/", true)
	if err != nil {
		log.Fatal("Cant get Parameter Store")
	}

}

type StackProps struct {
	StackProps awscdk.StackProps
}

var MANAGEMENT_PORT = aws.Float64(16686)

// resulting in SERVICE_NAME.NAMESPACE
var SERVICE_NAME = "jaeger"
var NAMESPACE = "otel.letsbuild-aws.com"
var HOSTED_ZONE_ID = "Z042038724KH99T9LFKK6"
var DNS_NAME = "service.letsbuild-aws.com"

func NewClusterStack(scope constructs.Construct, id string, props *StackProps) awscdk.Stack {
	var sprops awscdk.StackProps

	if props != nil {
		sprops = props.StackProps
	}
	stack := awscdk.NewStack(scope, &id, &sprops)
	this := stack

	// *********** VPC **************************************
	vpcid := ConfigurationParams.GetValueByName("vpcid")
	vpc := ec2.Vpc_FromLookup(stack, aws.String("basevpc"),
		&ec2.VpcLookupOptions{
			IsDefault: aws.Bool(false),
			VpcId:     &vpcid,
		})

	//*** security Group ************************************
	secGroup := ec2.NewSecurityGroup(this, aws.String("jaeger-sg"),
		&ec2.SecurityGroupProps{
			Vpc:               vpc,
			SecurityGroupName: aws.String("jaeger-sg"),
			AllowAllOutbound:  aws.Bool(true),
		})

	//**  Add IngressRule to access the docker image on 80 and 7070 ports
	secGroup.AddIngressRule(ec2.Peer_AnyIpv4(), ec2.Port_Tcp(aws.Float64(4317)), aws.String("otel_gprc"), aws.Bool(false))
	secGroup.AddIngressRule(ec2.Peer_AnyIpv4(), ec2.Port_Tcp(aws.Float64(4318)), aws.String("otel_http"), aws.Bool(false))
	secGroup.AddIngressRule(ec2.Peer_AnyIpv4(), ec2.Port_Tcp(MANAGEMENT_PORT), aws.String("GUI"), aws.Bool(false))

	// ** Execution Role ************************************
	excutionRole := awsiam.NewRole(this, aws.String("jaeger-exec-role"),
		&awsiam.RoleProps{
			AssumedBy:   awsiam.NewServicePrincipal(aws.String("ecs-tasks.amazonaws.com"), nil),
			Description: aws.String("Jaeger Execution Role"),
			Path:        aws.String("/otel/"),
			RoleName:    aws.String("jaeger-exec-role"),
		},
	)
	excutionRole.AddToPolicy(awsiam.NewPolicyStatement(
		&awsiam.PolicyStatementProps{
			Actions: &[]*string{
				aws.String("ecr:GetAuthorizationToken"),
				aws.String("ecr:BatchCheckLayerAvailability"),
				aws.String("ecr:GetDownloadUrlForLayer"),
				aws.String("ecr:BatchGetImage"),
				aws.String("logs:CreateLogStream"),
				aws.String("logs:PutLogEvents"),
			},
			Effect:    awsiam.Effect_ALLOW,
			Resources: &[]*string{aws.String("*")},
			Sid:       aws.String("ecrpuller"),
		},
	))

	// ** Task Role *****************************************
	taskRole := awsiam.NewRole(this, aws.String("jaeger-task-role"),
		&awsiam.RoleProps{
			AssumedBy:   awsiam.NewServicePrincipal(aws.String("ecs-tasks.amazonaws.com"), nil),
			Description: aws.String("Jaeger Execution Role"),
			Path:        aws.String("/otel/"),
			RoleName:    aws.String("jaeger-task-role"),
		},
	)
	taskRole.AddToPolicy(awsiam.NewPolicyStatement(
		&awsiam.PolicyStatementProps{
			Actions: &[]*string{
				aws.String("logs:CreateLogStream"),
				aws.String("logs:PutLogEvents"),
			},
			Effect:    awsiam.Effect_ALLOW,
			Resources: &[]*string{aws.String("*")},
			Sid:       aws.String("ecrpuller"),
		},
	))

	// *** Cluster ********************************************
	cluster := ecs.NewCluster(stack, aws.String("ECSCluster"), &ecs.ClusterProps{
		Vpc: vpc,
	})

	// *** task definition
	var task = ecs.NewFargateTaskDefinition(this, aws.String("JaegerAllService"),
		&ecs.FargateTaskDefinitionProps{
			ExecutionRole:       excutionRole,
			Family:              aws.String("megaproaktiv-jaeger-new"),
			TaskRole:            taskRole,
			Cpu:                 aws.Float64(2048),
			EphemeralStorageGiB: new(float64),
			MemoryLimitMiB:      aws.Float64(4096),
		},
	)

	// See ports
	// https://www.jaegertracing.io/docs/1.18/deployment/#query-service--ui
	task.AddContainer(aws.String("jaegerContainer"),
		&ecs.ContainerDefinitionOptions{
			Image:         ecs.ContainerImage_FromRegistry(aws.String("jaegertracing/all-in-one:1.39.0"), nil),
			ContainerName: aws.String("jaeger-all"),
			Essential:     aws.Bool(true),
			// See https://aws.amazon.com/de/blogs/containers/graceful-shutdowns-with-ecs/
			StopTimeout: awscdk.Duration_Seconds(aws.Float64(5)),
			Environment: &map[string]*string{
				"SPAN_STORAGE_TYPE":      aws.String("memory"),
				"COLLECTOR_OTLP_ENABLED": aws.String("true"),
				"LOG_LEVEL":              aws.String("debug"),
			},
			Logging: ecs.AwsLogDriver_AwsLogs(&ecs.AwsLogDriverProps{
				StreamPrefix: aws.String("jaeger"),
				// LogGroup:         nil,
				LogRetention: awslogs.RetentionDays_ONE_MONTH,
				Mode:         ecs.AwsLogDriverMode_NON_BLOCKING,
				// MultilinePattern: new(string),
			}),
			MemoryLimitMiB:       aws.Float64(4096),
			MemoryReservationMiB: aws.Float64(2048),
			// The first port is used for the listener-target configuration
			PortMappings: &[]*ecs.PortMapping{
				{
					ContainerPort: MANAGEMENT_PORT,
					HostPort:      MANAGEMENT_PORT,
					Protocol:      ecs.Protocol_TCP,
					// management
				},
				{
					ContainerPort: aws.Float64(4317),
					HostPort:      aws.Float64(4317),
					Protocol:      ecs.Protocol_TCP,
					// "otel-grpc"
				},
				{
					ContainerPort: aws.Float64(4318),
					HostPort:      aws.Float64(4318),
					Protocol:      ecs.Protocol_TCP,
					// "otel-http",
				},
				{
					ContainerPort: aws.Float64(14268),
					HostPort:      aws.Float64(14268),
					Protocol:      ecs.Protocol_TCP,
					// jaeger_thrift
				},
				{
					ContainerPort: aws.Float64(14250),
					HostPort:      aws.Float64(14250),
					Protocol:      ecs.Protocol_TCP,
					// model_proto
				},
			},
		},
	)

	namespace := awsservicediscovery.NewPrivateDnsNamespace(this, aws.String("oteltrace-namespace"),
		&awsservicediscovery.PrivateDnsNamespaceProps{
			Name:        aws.String(NAMESPACE),
			Description: aws.String("DNS service discovery subdomain"),
			Vpc:         vpc,
		},
	)

	// *** Fargate Service from cluster, task definition and the security group
	// *** !! If used in public subnet, you have to set **AssignPublicIp**
	fargateService := ecs.NewFargateService(this, aws.String("jaeger-fargate-service"),
		&ecs.FargateServiceProps{
			Cluster:              cluster,
			DesiredCount:         aws.Float64(1),
			EnableExecuteCommand: aws.Bool(true),
			ServiceName:          aws.String("jaeger-service"),
			CloudMapOptions: &ecs.CloudMapOptions{
				CloudMapNamespace: namespace,
				ContainerPort:     aws.Float64(4317),
				DnsRecordType:     awsservicediscovery.DnsRecordType_A,
				Name:              &SERVICE_NAME,
			},
			TaskDefinition: task,
			SecurityGroups: &[]ec2.ISecurityGroup{
				secGroup,
			},
			VpcSubnets: &ec2.SubnetSelection{
				SubnetType: ec2.SubnetType_PRIVATE_WITH_EGRESS,
			},
		})

	// ***  ALB using the above VPC to connect to the GUI from outside
	lb := elbv2.NewApplicationLoadBalancer(this, aws.String("nlb-jaeger"),
		&elbv2.ApplicationLoadBalancerProps{
			Vpc:              vpc,
			InternetFacing:   aws.Bool(true),
			LoadBalancerName: aws.String("jaeger"),
			VpcSubnets: &ec2.SubnetSelection{
				SubnetType: ec2.SubnetType_PUBLIC,
			},
		},
	)

	//Port: MANAGEMENT_PORT,
	//** Listeners **
	// Certificate
	hostedZoneId := HOSTED_ZONE_ID
	hostedZone := awsroute53.HostedZone_FromHostedZoneAttributes(this, aws.String("zoneAttributes"),
		&awsroute53.HostedZoneAttributes{
			HostedZoneId: &hostedZoneId,
			ZoneName:     &DNS_NAME,
		})

	certificate := awscertificatemanager.NewDnsValidatedCertificate(this, aws.String("certificate"),
		&awscertificatemanager.DnsValidatedCertificateProps{
			DomainName: &DNS_NAME,
			HostedZone: hostedZone,
			Region:     this.Region(),
			Validation: awscertificatemanager.CertificateValidation_FromDns(hostedZone),
		},
	)

	// DNS Entry
	awsroute53.NewARecord(this, aws.String("jaegerarecord"),
		&awsroute53.ARecordProps{
			Zone:           hostedZone,
			DeleteExisting: aws.Bool(true),
			RecordName:     &DNS_NAME,
			Target:         awsroute53.RecordTarget_FromAlias(awsroute53targets.NewLoadBalancerTarget(lb)),
		},
	)

	listenerGui := lb.AddListener(aws.String("jaeger-listener1"),
		&elbv2.BaseApplicationListenerProps{
			Port:     aws.Float64(443),
			Protocol: elbv2.ApplicationProtocol_HTTPS,
			Certificates: &[]elbv2.IListenerCertificate{
				certificate,
			},
		},
	)


	listenerGui.AddTargets(aws.String("jaegerservice"),
		&elbv2.AddApplicationTargetsProps{
			Port:                MANAGEMENT_PORT,
			Protocol:            elbv2.ApplicationProtocol_HTTP,
			DeregistrationDelay: awscdk.Duration_Seconds(aws.Float64(5)),
			HealthCheck:         &elbv2.HealthCheck{},
			TargetGroupName:     aws.String("otel"),
			Targets: &[]elbv2.IApplicationLoadBalancerTarget{
				fargateService,
			},
		},
	)


	adminUrl := "https://" + DNS_NAME

	awsssm.NewStringParameter(this, aws.String("nlpipoutput"),
		&awsssm.StringParameterProps{
			ParameterName: aws.String("/otel/jaeger/ip"),
			StringValue:   lb.LoadBalancerDnsName(),
		},
	)

	awscdk.NewCfnOutput(this, aws.String("nlbnameoutput"),
		&awscdk.CfnOutputProps{
			Value:       &adminUrl,
			Description: aws.String("DNS name of the network load balancer for jaeger service"),
			ExportName:  aws.String("jaegeradmin"),
		},
	)

	return stack
}
