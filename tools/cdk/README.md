# Fixture collector

This is an [AWS CDK](https://docs.aws.amazon.com/cdk/v2/guide/home.html) app
that deploys ECS resources to AWS to enable real [task metadata
API](https://docs.aws.amazon.com/AmazonECS/latest/developerguide/task-metadata-endpoint.html)
responses to be collected and versioned in this repository as test fixtures.
It's important to power our tests using real fixtures, as the documentation of
the task metadata API is not comprehensive or even perfectly accurate, and the
API is subject to change at any time.

The expectation is that we collect sufficient fixtures to exercise all the
features of the exporter and all the hidden edge cases of the task metadata API
with respect to task configuration. So, for example, we know that ECS on EC2 and
on Fargate use completely different implementations of the API, so we should
deploy all tasks to both in order to collect fixtures from both.

## How to update fixtures

### Prerequisites

You need to follow the [AWS CDK setup
guide](https://docs.aws.amazon.com/cdk/v2/guide/prerequisites.html). The full
details are there, but in short, you will need:
- NodeJS installed, with the `aws-cdk` NPM package globally installed. CDK is
  written in Node. Even though our CDK app is written in Go, any non-NodeJS CDK
  app is ultimately doing RPC to a NodeJS process using code generated from the
  CDK NodeJS codebase.
- An AWS account,
  [bootstrapped](https://docs.aws.amazon.com/cdk/v2/guide/bootstrapping-env.html)
  to receive CDK deployments.
- The AWS CLI installed, with proper credentials for that account configured.
  You also need the extra [Session Manager
  plugin](https://docs.aws.amazon.com/AmazonECS/latest/developerguide/ecs-exec.html)
  for the CLI, to enable running commands inside task containers to actually
  download the fixtures, using the CLI.

Additionally, the update steps below use `jq` to query and pretty print JSON
output.

### Update steps

With all that done, the process works as follows:
1. Deploy our CDK app's stack, which will result in various ECS tasks being
   launched.
2. Run commands in said tasks to produce the fixtures.
3. Destroy the stack, such that you are no longer paying money to AWS.

In other words:

```sh
# Prerequisite: you are authenticated to your AWS account. This can be done in
# multiple ways; one common way involves having an AWS_ACCESS_KEY_ID and
# AWS_SECRET_ACCESS_KEY set in the environment.

# Deploy the stack. You will have to confirm the changes interactively.
cdk deploy

# Update fixtures. We have to use `tail`/`head` to chop off non-JSON output
# printed by `aws ecs execute-command` to stdout. See:
# https://github.com/aws/session-manager-plugin/issues/85
#
# We also use `jq` to sort parts of the output data to keep fixture diffs more
# readable - some things are not consistently ordered.
aws ecs execute-command --interactive --cluster prom-ecs-exporter-fixtures --task "$(aws ecs list-tasks --cluster prom-ecs-exporter-fixtures --service prom-ecs-exporter-fixtures-fargate | jq -r .taskArns[0])" --container ecs-exporter --command 'sh -c "wget -q -O- ${ECS_CONTAINER_METADATA_URI_V4}/task"' | tail -n4 | head -n1 | jq '.Containers |= sort_by(.Name)' > ../../ecscollector/testdata/fixtures/fargate_task_metadata.json
aws ecs execute-command --interactive --cluster prom-ecs-exporter-fixtures --task "$(aws ecs list-tasks --cluster prom-ecs-exporter-fixtures --service prom-ecs-exporter-fixtures-fargate | jq -r .taskArns[0])" --container ecs-exporter --command 'sh -c "wget -q -O- ${ECS_CONTAINER_METADATA_URI_V4}/task/stats"' | tail -n4 | head -n1 | jq 'to_entries | sort_by(.value.name) | from_entries' > ../../ecscollector/testdata/fixtures/fargate_task_stats.json
aws ecs execute-command --interactive --cluster prom-ecs-exporter-fixtures --task "$(aws ecs list-tasks --cluster prom-ecs-exporter-fixtures --service prom-ecs-exporter-fixtures-ec2 | jq -r .taskArns[0])" --container ecs-exporter --command 'sh -c "wget -q -O- ${ECS_CONTAINER_METADATA_URI_V4}/task"' | tail -n4 | head -n1 | jq '.Containers |= sort_by(.Name)' > ../../ecscollector/testdata/fixtures/ec2_task_metadata.json
aws ecs execute-command --interactive --cluster prom-ecs-exporter-fixtures --task "$(aws ecs list-tasks --cluster prom-ecs-exporter-fixtures --service prom-ecs-exporter-fixtures-ec2 | jq -r .taskArns[0])" --container ecs-exporter --command 'sh -c "wget -q -O- ${ECS_CONTAINER_METADATA_URI_V4}/task/stats"' | tail -n4 | head -n1 | jq 'to_entries | sort_by(.value.name) | from_entries' > ../../ecscollector/testdata/fixtures/ec2_task_stats.json

# Destroy the stack. You will have to confirm the changes interactively.
cdk destroy
```
