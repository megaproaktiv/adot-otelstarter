# Open Telemetry with Lambda

- Using adot

## Structure

### app

The lambda application

### Infra

The Lambda infrastructure

### Test 

Overall test

## jaeger

Jaeger server

See the blog post for details:

## Walktrough


1. Clone repository

```bash
git clone https://github.com/megaproaktiv/adot-otelstarter.git
```

2. Set region
  export AWS_REGION=yourregion,  e.g. 
```bash  
  export AWS_REGION=eu-central-1
```

3. If CDK is not bootstrapped:

```bash  
  task bootstrap
```

4. Create VPC

```bash  
  task jaeger:deploy-vpc
```
  
5. Set Domain and Service configuration

Edit   `jaeger/cluster.go`:  

```bash  
  var SERVICE_NAME = "jaeger"
  var NAMESPACE = "otel.letsbuild-aws.com"
  var HOSTED_ZONE_ID = "Z042035555KH99T9LFKK6"
  var DNS_NAME = "service.letsbuild-aws.com"
```

6. Create ECS cluster with jaeger service

```bash  
  task jaeger:deploy-jaeger
```

7. Deploy Lambda Resources and function
  
```bash  
  task deploy
```

Note: because of the ENI this could take a few minutes

8. Create Traffic

```bash  
  ./test/traffic.sh
```
  

Now access the jaeger UI via the ALB ip or with the configured domain.  