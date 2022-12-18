#!/bin/bash
export BUCKET=`aws cloudformation describe-stacks --stack-name otelstarter --query "Stacks[?StackName == 'otelstarter'][].Outputs[?OutputKey == 'Bucket'].OutputValue" --output text`
for i in 0 1 2 3 4 5 6 7 8 9 
do
    for k in 0 1 2 3 4 5 6 7 8 9
    do
        for l in 0 1 2 3 4 5 6 7 8 9
        do
            date
            sleep $[ ( $RANDOM % 10 )  ]
            aws s3 cp readme.md s3://${BUCKET}/test-2${i}-${k}
        done
    done
done
