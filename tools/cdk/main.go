// Copyright 2025 The Prometheus Authors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"github.com/aws/aws-cdk-go/awscdk/v2"
	"github.com/aws/aws-cdk-go/awscdk/v2/awsautoscaling"
	"github.com/aws/aws-cdk-go/awscdk/v2/awsec2"
	"github.com/aws/aws-cdk-go/awscdk/v2/awsecs"
	"github.com/aws/aws-cdk-go/awscdk/v2/customresources"
	"github.com/aws/constructs-go/constructs/v10"
	"github.com/aws/jsii-runtime-go"
)

type TetStackProps struct {
	awscdk.StackProps
}

func main() {
	defer jsii.Close()
	app := awscdk.NewApp(nil)
	_ = NewFixtureCollectorStack(app, "FixtureCollectorStack")
	app.Synth(nil)
}

func NewFixtureCollectorStack(scope constructs.Construct, id string) awscdk.Stack {
	stack := awscdk.NewStack(scope, &id, &awscdk.StackProps{})

	// The human-readable name for many resources we create here.
	resourceName := "prom-ecs-exporter-fixtures"

	vpc := awsec2.NewVpc(stack, jsii.String("Vpc"), &awsec2.VpcProps{MaxAzs: jsii.Number(1)})
	cluster := awsecs.NewCluster(stack, jsii.String("Cluster"), &awsecs.ClusterProps{
		ClusterName:                    &resourceName,
		Vpc:                            vpc,
		EnableFargateCapacityProviders: jsii.Bool(true),
	})

	// ASG capacity provider used for tasks on EC2 instances.
	autoScalingGroup := awsautoscaling.NewAutoScalingGroup(stack, jsii.String("ASG"), &awsautoscaling.AutoScalingGroupProps{
		Vpc:                  vpc,
		AutoScalingGroupName: &resourceName,
		InstanceType:         awsec2.NewInstanceType(jsii.String("t4g.nano")),
		MachineImage:         awsecs.EcsOptimizedImage_AmazonLinux2023(awsecs.AmiHardwareType_ARM, nil),
		MinCapacity:          jsii.Number(0),
		MaxCapacity:          jsii.Number(2),
		BlockDevices: &[]*awsautoscaling.BlockDevice{
			{DeviceName: jsii.String("/dev/xvda"), Volume: awsautoscaling.BlockDeviceVolume_Ebs(jsii.Number(40), nil)},
		},
		NewInstancesProtectedFromScaleIn: jsii.Bool(true),
	})
	// https://github.com/aws/aws-cdk/issues/18179#issuecomment-1150981559
	customresources.NewAwsCustomResource(stack, jsii.String("AsgForceDelete"), &customresources.AwsCustomResourceProps{
		OnDelete: &customresources.AwsSdkCall{
			Service: jsii.String("AutoScaling"),
			Action:  jsii.String("deleteAutoScalingGroup"),
			Parameters: map[string]any{
				"AutoScalingGroupName": autoScalingGroup.AutoScalingGroupName(),
				"ForceDelete":          jsii.Bool(true),
			},
		},
		Policy: customresources.AwsCustomResourcePolicy_FromSdkCalls(&customresources.SdkCallsPolicyOptions{
			Resources: customresources.AwsCustomResourcePolicy_ANY_RESOURCE(),
		}),
	}).Node().AddDependency(autoScalingGroup)

	capacityProvider := awsecs.NewAsgCapacityProvider(stack, jsii.String("Capacity"), &awsecs.AsgCapacityProviderProps{
		CapacityProviderName:               &resourceName,
		InstanceWarmupPeriod:               jsii.Number(0),
		EnableManagedTerminationProtection: jsii.Bool(true),
		AutoScalingGroup:                   autoScalingGroup,
	})
	cluster.AddAsgCapacityProvider(capacityProvider, nil)

	// We do not strictly need such an image to capture the fixtures we need,
	// but given that there should always be an ecs-exporter sidecar in every
	// real Task, it would be strange not to include in the fixture data here.
	ecsExporterImage := awsecs.ContainerImage_FromRegistry(jsii.String("quay.io/prometheuscommunity/ecs-exporter:v0.4.0"), nil)

	{
		// Create an EC2 task.
		taskDefinition := awsecs.NewTaskDefinition(stack, jsii.String("Ec2TaskDefinition"), &awsecs.TaskDefinitionProps{
			Family:        jsii.String("ecs-exporter-fixtures-ec2"),
			NetworkMode:   awsecs.NetworkMode_AWS_VPC,
			Compatibility: awsecs.Compatibility_EC2,
			MemoryMiB:     jsii.String("256"),
		})
		taskDefinition.AddContainer(jsii.String("EcsExporter"), &awsecs.ContainerDefinitionOptions{
			ContainerName:        jsii.String("ecs-exporter"),
			Image:                ecsExporterImage,
			MemoryReservationMiB: jsii.Number(128),
			MemoryLimitMiB:       jsii.Number(128),
			Cpu:                  jsii.Number(128),
		})
		taskDefinition.AddContainer(jsii.String("AlpineShell"), &awsecs.ContainerDefinitionOptions{
			ContainerName: jsii.String("main"),
			Image:         awsecs.ContainerImage_FromRegistry(jsii.String("alpine"), nil),
			// Hang on forever.
			Command: &[]*string{jsii.String("sh"), jsii.String("-c"), jsii.String("sleep infinity")},
		})
		taskDefinition.AddContainer(jsii.String("Nonessential"), &awsecs.ContainerDefinitionOptions{
			ContainerName:        jsii.String("nonessential"),
			Image:                awsecs.ContainerImage_FromRegistry(jsii.String("alpine"), nil),
			Command:              &[]*string{jsii.String("sh"), jsii.String("-c"), jsii.String("echo goodbye")},
			MemoryReservationMiB: jsii.Number(128),
			MemoryLimitMiB:       jsii.Number(128),
			Cpu:                  jsii.Number(128),
			Essential:            jsii.Bool(false),
		})

		service := awsecs.NewEc2Service(stack, jsii.String("Ec2Service"), &awsecs.Ec2ServiceProps{
			ServiceName:       jsii.String(resourceName + "-ec2"),
			Cluster:           cluster,
			TaskDefinition:    taskDefinition,
			DesiredCount:      jsii.Number(1),
			MinHealthyPercent: jsii.Number(0),
			CapacityProviderStrategies: &[]*awsecs.CapacityProviderStrategy{
				{CapacityProvider: capacityProvider.CapacityProviderName(), Weight: jsii.Number(1)},
			},
			EnableExecuteCommand: jsii.Bool(true),
		})
		// Deletion does not work if we don't do this - the VPC will tear down
		// route tables etc before we can delete the cluster/capacity provider,
		// leaving instances and their tasks in limbo.
		vpcConstructs := *vpc.Node().FindAll(constructs.ConstructOrder_PREORDER)
		vpcDependables := make([]constructs.IDependable, len(vpcConstructs))
		for i, c := range vpcConstructs {
			vpcDependables[i] = c
		}
		service.Node().AddDependency(vpcDependables...)
	}

	{
		// Create a Fargate task.
		taskDefinition := awsecs.NewFargateTaskDefinition(stack, jsii.String("FargateTaskDefinition"), &awsecs.FargateTaskDefinitionProps{
			Family: jsii.String("ecs-exporter-fixtures-fargate"),
			RuntimePlatform: &awsecs.RuntimePlatform{
				CpuArchitecture:       awsecs.CpuArchitecture_ARM64(),
				OperatingSystemFamily: awsecs.OperatingSystemFamily_LINUX(),
			},
		})
		taskDefinition.AddContainer(jsii.String("EcsExporter"), &awsecs.ContainerDefinitionOptions{
			ContainerName:        jsii.String("ecs-exporter"),
			Image:                ecsExporterImage,
			MemoryReservationMiB: jsii.Number(128),
			MemoryLimitMiB:       jsii.Number(128),
			Cpu:                  jsii.Number(128),
		})
		taskDefinition.AddContainer(jsii.String("AlpineShell"), &awsecs.ContainerDefinitionOptions{
			ContainerName: jsii.String("main"),
			Image:         awsecs.ContainerImage_FromRegistry(jsii.String("alpine"), nil),
			// Hang on forever.
			Command: &[]*string{jsii.String("sh"), jsii.String("-c"), jsii.String("sleep infinity")},
		})
		taskDefinition.AddContainer(jsii.String("Nonessential"), &awsecs.ContainerDefinitionOptions{
			ContainerName:        jsii.String("nonessential"),
			Image:                awsecs.ContainerImage_FromRegistry(jsii.String("alpine"), nil),
			Command:              &[]*string{jsii.String("sh"), jsii.String("-c"), jsii.String("echo goodbye")},
			MemoryReservationMiB: jsii.Number(128),
			MemoryLimitMiB:       jsii.Number(128),
			Cpu:                  jsii.Number(128),
			Essential:            jsii.Bool(false),
		})

		awsecs.NewFargateService(stack, jsii.String("FargateService"), &awsecs.FargateServiceProps{
			ServiceName:          jsii.String(resourceName + "-fargate"),
			Cluster:              cluster,
			TaskDefinition:       taskDefinition,
			DesiredCount:         jsii.Number(1),
			MinHealthyPercent:    jsii.Number(0),
			EnableExecuteCommand: jsii.Bool(true),
		})
	}

	awscdk.Aspects_Of(stack).Add(&CapacityProviderDependencyAspect{}, nil)

	return stack
}

type CapacityProviderDependencyAspect struct{}

// Add a dependency from capacity provider association to the cluster
// and from each service to the capacity provider association.
//
// https://github.com/aws/aws-cdk/issues/19275#issuecomment-1152860147
func (CapacityProviderDependencyAspect) Visit(node constructs.IConstruct) {
	if service, ok := node.(awsecs.Ec2Service); ok {
		for _, child := range *service.Cluster().Node().FindAll(constructs.ConstructOrder_PREORDER) {
			if assoc, ok := child.(awsecs.CfnClusterCapacityProviderAssociations); ok {
				assoc.Node().AddDependency(service.Cluster())
				service.Node().AddDependency(assoc)
			}
		}
	}
}
