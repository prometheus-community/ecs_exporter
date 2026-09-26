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

The `main` container uses a standard-library-only Go workload built from
[`fixture-workload`](./fixture-workload). It creates file-cache pressure and
retains shared, transparent-huge-page, locked, socket-buffer, and kernel-stack
memory so the task stats fixtures exercise obscure aspects of cgroup memory
accounting rather than only describing an idle container. EC2 instances enable
the kernel's `madvise` transparent huge-page policy, and the workload verifies
and maintains its advised huge-page mapping after inducing memory pressure.

## How to update fixtures

### Prerequisites

You need to follow the [AWS CDK setup
guide](https://docs.aws.amazon.com/cdk/v2/guide/prerequisites.html). The full
details are there, but in short, you will need:
- NodeJS installed, with the `aws-cdk` NPM package globally installed. CDK is
  written in Node. Even though our CDK app is written in Go, any non-NodeJS CDK
  app is ultimately doing RPC to a NodeJS process using code generated from the
  CDK NodeJS codebase.
- Docker with BuildKit support, used to cross-compile (if necessary) and publish
  the fixture workload image. The build itself runs on the host architecture
  and does not require emulation.
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
# multiple ways; one common way involves using `aws login` from the AWS CLI.

# Deploy the stack.
cdk deploy -y

# Update fixtures. We select the JSON line from the additional output printed
# by `aws ecs execute-command`. See:
# https://github.com/aws/session-manager-plugin/issues/85
#
# We also use `jq` to sort parts of the output data to keep fixture diffs more
# readable - some things are not consistently ordered.
set -o pipefail
aws ecs execute-command --interactive --cluster prom-ecs-exporter-fixtures --task "$(aws ecs list-tasks --cluster prom-ecs-exporter-fixtures --service prom-ecs-exporter-fixtures-fargate | jq -r .taskArns[0])" --container ecs-exporter --command 'sh -c "wget -q -O- ${ECS_CONTAINER_METADATA_URI_V4}/task; echo"' 2>&1 | sed -n '/^{/p' | head -n1 | jq -e '.Containers |= sort_by(.Name)' > ../../ecscollector/testdata/fixtures/fargate_task_metadata.json.tmp && mv ../../ecscollector/testdata/fixtures/fargate_task_metadata.json.tmp ../../ecscollector/testdata/fixtures/fargate_task_metadata.json
aws ecs execute-command --interactive --cluster prom-ecs-exporter-fixtures --task "$(aws ecs list-tasks --cluster prom-ecs-exporter-fixtures --service prom-ecs-exporter-fixtures-fargate | jq -r .taskArns[0])" --container ecs-exporter --command 'sh -c "wget -q -O- ${ECS_CONTAINER_METADATA_URI_V4}/task/stats; echo"' 2>&1 | sed -n '/^{/p' | head -n1 | jq -e 'to_entries | sort_by(.value.name) | from_entries' > ../../ecscollector/testdata/fixtures/fargate_task_stats.json.tmp && mv ../../ecscollector/testdata/fixtures/fargate_task_stats.json.tmp ../../ecscollector/testdata/fixtures/fargate_task_stats.json
aws ecs execute-command --interactive --cluster prom-ecs-exporter-fixtures --task "$(aws ecs list-tasks --cluster prom-ecs-exporter-fixtures --service prom-ecs-exporter-fixtures-ec2 | jq -r .taskArns[0])" --container ecs-exporter --command 'sh -c "wget -q -O- ${ECS_CONTAINER_METADATA_URI_V4}/task; echo"' 2>&1 | sed -n '/^{/p' | head -n1 | jq -e '.Containers |= sort_by(.Name)' > ../../ecscollector/testdata/fixtures/ec2_task_metadata.json.tmp && mv ../../ecscollector/testdata/fixtures/ec2_task_metadata.json.tmp ../../ecscollector/testdata/fixtures/ec2_task_metadata.json
aws ecs execute-command --interactive --cluster prom-ecs-exporter-fixtures --task "$(aws ecs list-tasks --cluster prom-ecs-exporter-fixtures --service prom-ecs-exporter-fixtures-ec2 | jq -r .taskArns[0])" --container ecs-exporter --command 'sh -c "wget -q -O- ${ECS_CONTAINER_METADATA_URI_V4}/task/stats; echo"' 2>&1 | sed -n '/^{/p' | head -n1 | jq -e 'to_entries | sort_by(.value.name) | from_entries' > ../../ecscollector/testdata/fixtures/ec2_task_stats.json.tmp && mv ../../ecscollector/testdata/fixtures/ec2_task_stats.json.tmp ../../ecscollector/testdata/fixtures/ec2_task_stats.json

# Verify that the bounded startup workload completed and exercised the memory
# accounting that the fixtures are intended to cover.
jq -e '
  .Containers[] | select(.Name == "main") |
  .KnownStatus == "RUNNING" and .Health.status == "HEALTHY"
' ../../ecscollector/testdata/fixtures/ec2_task_metadata.json
jq -e --arg id "$(jq -r '.Containers[] | select(.Name == "main") | .DockerId' ../../ecscollector/testdata/fixtures/ec2_task_metadata.json)" '
  .[$id].memory_stats as $memory |
  $memory.limit == 100663296 and
  (($memory.failcnt // 0) == 0) and
  $memory.stats.anon_thp > 0 and
  $memory.stats.kernel_stack > 0 and
  $memory.stats.sock > 0 and
  $memory.stats.shmem > 0 and
  $memory.stats.unevictable > 0 and
  $memory.stats.slab_reclaimable > 0 and
  $memory.stats.slab_unreclaimable > 0 and
  $memory.stats.pglazyfree > 0 and
  $memory.stats.pglazyfreed > 0 and
  $memory.stats.pgscan > 0 and
  $memory.stats.pgsteal > 0
' ../../ecscollector/testdata/fixtures/ec2_task_stats.json

jq -e '
  .Containers[] | select(.Name == "main") |
  .KnownStatus == "RUNNING" and .Health.status == "HEALTHY"
' ../../ecscollector/testdata/fixtures/fargate_task_metadata.json
jq -e --arg id "$(jq -r '.Containers[] | select(.Name == "main") | .DockerId' ../../ecscollector/testdata/fixtures/fargate_task_metadata.json)" '
  .[$id].memory_stats as $memory |
  $memory.limit == 100663296 and
  $memory.failcnt > 0 and
  $memory.max_usage > 0 and
  $memory.stats.total_rss_huge > 0 and
  $memory.stats.total_unevictable > 0 and
  $memory.stats.total_cache > 0 and
  $memory.stats.total_pgpgin > 0 and
  $memory.stats.total_pgpgout > 0
' ../../ecscollector/testdata/fixtures/fargate_task_stats.json

# Destroy the stack.
cdk destroy -y
```
