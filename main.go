package main

import (
	"github.com/pulumi/pulumi-aws/sdk/v6/go/aws/ec2"
	"github.com/pulumi/pulumi-aws/sdk/v6/go/aws/ec2transitgateway"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {
		// Create egress VPC
		egressVpc, err := ec2.NewVpc(ctx, "egress-vpc", &ec2.VpcArgs{
			CidrBlock: pulumi.String("10.0.0.0/16"),
		})
		if err != nil {
			return err
		}
		publicSubnet1, err := ec2.NewSubnet(ctx, "public-subnet-1", &ec2.SubnetArgs{
			CidrBlock:        pulumi.String("10.0.1.0/24"),
			VpcId:            egressVpc.ID(),
			AvailabilityZone: pulumi.String("eu-north-1a"),
		})
		if err != nil {
			return err
		}
		publicSubnet2, err := ec2.NewSubnet(ctx, "public-subnet-2", &ec2.SubnetArgs{
			CidrBlock:        pulumi.String("10.0.2.0/24"),
			VpcId:            egressVpc.ID(),
			AvailabilityZone: pulumi.String("eu-north-1b"),
		})
		if err != nil {
			return err
		}
		igw, err := ec2.NewInternetGateway(ctx, "igw", &ec2.InternetGatewayArgs{
			VpcId: egressVpc.ID(),
		})
		if err != nil {
			return err
		}

		natIP1, err := ec2.NewEip(ctx, "nat-ip-1", &ec2.EipArgs{})
		if err != nil {
			return err
		}
		publicNatGW1, err := ec2.NewNatGateway(ctx, "nat-gateway-subnet1", &ec2.NatGatewayArgs{
			SubnetId:     publicSubnet1.ID(),
			AllocationId: natIP1.ID(),
		})
		if err != nil {
			return err
		}
		natIP2, err := ec2.NewEip(ctx, "nat-ip-2", &ec2.EipArgs{})
		if err != nil {
			return err
		}
		publicNatGW2, err := ec2.NewNatGateway(ctx, "nat-gateway-subnet2", &ec2.NatGatewayArgs{
			SubnetId:     publicSubnet2.ID(),
			AllocationId: natIP2.ID(),
		})
		if err != nil {
			return err
		}

		privateEgressSubnet1, err := ec2.NewSubnet(ctx, "private-egress-subnet-1", &ec2.SubnetArgs{
			CidrBlock:        pulumi.String("10.0.3.0/24"),
			VpcId:            egressVpc.ID(),
			AvailabilityZone: pulumi.String("eu-north-1a"),
		})
		if err != nil {
			return err
		}
		privateEgressSubnet2, err := ec2.NewSubnet(ctx, "private-egress-subnet-2", &ec2.SubnetArgs{
			CidrBlock:        pulumi.String("10.0.4.0/24"),
			VpcId:            egressVpc.ID(),
			AvailabilityZone: pulumi.String("eu-north-1b"),
		})
		if err != nil {
			return err
		}

		privateEgressRT1, err := ec2.NewRouteTable(ctx, "private-egress-route-table-1", &ec2.RouteTableArgs{
			Tags: pulumi.StringMap{
				"Name": pulumi.String("egress-vpc-private-rt1"),
			},
			VpcId: egressVpc.ID(),
			Routes: ec2.RouteTableRouteArray{
				&ec2.RouteTableRouteArgs{
					CidrBlock:    pulumi.String("0.0.0.0/0"),
					NatGatewayId: publicNatGW1.ID(),
				},
			},
		})
		if err != nil {
			return err
		}

		privateEgressRT2, err := ec2.NewRouteTable(ctx, "private-egress-route-table-2", &ec2.RouteTableArgs{
			Tags: pulumi.StringMap{
				"Name": pulumi.String("egress-vpc-private-rt2"),
			},
			VpcId: egressVpc.ID(),
			Routes: ec2.RouteTableRouteArray{
				&ec2.RouteTableRouteArgs{
					CidrBlock:    pulumi.String("0.0.0.0/0"),
					NatGatewayId: publicNatGW2.ID(),
				},
			},
		})
		if err != nil {
			return err
		}

		_, err = ec2.NewRouteTableAssociation(ctx, "private-egress-subnet-1-association", &ec2.RouteTableAssociationArgs{
			SubnetId:     privateEgressSubnet1.ID(),
			RouteTableId: privateEgressRT1.ID(),
		})
		if err != nil {
			return err
		}
		_, err = ec2.NewRouteTableAssociation(ctx, "private-egress-subnet-2-association", &ec2.RouteTableAssociationArgs{
			SubnetId:     privateEgressSubnet2.ID(),
			RouteTableId: privateEgressRT2.ID(),
		})
		if err != nil {
			return err
		}

		// Create transit GW
		tgw, err := ec2transitgateway.NewTransitGateway(ctx, "transit-gateway", &ec2transitgateway.TransitGatewayArgs{})
		if err != nil {
			return err
		}

		internetTGWRouteTable, err := ec2transitgateway.NewRouteTable(ctx, "transit-gateway-route-table", &ec2transitgateway.RouteTableArgs{
			TransitGatewayId: tgw.ID(),
		})
		if err != nil {
			return err
		}

		egressVPCAttachment, err := ec2transitgateway.NewVpcAttachment(ctx, "egress-vpc-attachment", &ec2transitgateway.VpcAttachmentArgs{
			SubnetIds: pulumi.StringArray{
				privateEgressSubnet1.ID(),
				privateEgressSubnet2.ID(),
			},
			TransitGatewayId: tgw.ID(),
			VpcId:            egressVpc.ID(),
		})
		if err != nil {
			return err
		}

		_, err = ec2transitgateway.NewRoute(ctx, "transit-gateway-route", &ec2transitgateway.RouteArgs{
			DestinationCidrBlock:       pulumi.String("0.0.0.0/0"),
			TransitGatewayAttachmentId: egressVPCAttachment.ID(),
			TransitGatewayRouteTableId: internetTGWRouteTable.ID(),
		})
		if err != nil {
			return err
		}

		_, err = ec2transitgateway.NewRouteTableAssociation(ctx, "transit-gateway-route-table-association-egress-vpc", &ec2transitgateway.RouteTableAssociationArgs{
			TransitGatewayAttachmentId: egressVPCAttachment.ID(),
			TransitGatewayRouteTableId: internetTGWRouteTable.ID(),
			ReplaceExistingAssociation: pulumi.Bool(true),
		})
		if err != nil {
			return err
		}
		_, err = ec2transitgateway.NewRouteTablePropagation(ctx, "transit-gateway-route-table-propagation-egress-vpc", &ec2transitgateway.RouteTablePropagationArgs{
			TransitGatewayAttachmentId: egressVPCAttachment.ID(),
			TransitGatewayRouteTableId: internetTGWRouteTable.ID(),
		})
		if err != nil {
			return err
		}

		// Connect compute VPC to transit GW for internet access
		computeVPC, err := ec2.NewVpc(ctx, "compute-vpc", &ec2.VpcArgs{
			CidrBlock:          pulumi.String("10.1.0.0/16"),
			EnableDnsHostnames: pulumi.Bool(true),
			EnableDnsSupport:   pulumi.Bool(true),
		})
		if err != nil {
			return err
		}
		computeSubnet1, err := ec2.NewSubnet(ctx, "compute-subnet-1", &ec2.SubnetArgs{
			CidrBlock:        pulumi.String("10.1.1.0/24"),
			VpcId:            computeVPC.ID(),
			AvailabilityZone: pulumi.String("eu-north-1a"),
		})
		if err != nil {
			return err
		}
		computeSubnet2, err := ec2.NewSubnet(ctx, "compute-subnet-2", &ec2.SubnetArgs{
			CidrBlock:        pulumi.String("10.1.2.0/24"),
			VpcId:            computeVPC.ID(),
			AvailabilityZone: pulumi.String("eu-north-1b"),
		})
		if err != nil {
			return err
		}
		computeRT, err := ec2.NewRouteTable(ctx, "compute-route-table", &ec2.RouteTableArgs{
			VpcId: computeVPC.ID(),
			Routes: ec2.RouteTableRouteArray{
				&ec2.RouteTableRouteArgs{
					CidrBlock:        pulumi.String("0.0.0.0/0"),
					TransitGatewayId: tgw.ID(),
				},
			},
		})
		if err != nil {
			return err
		}
		_, err = ec2.NewRouteTableAssociation(ctx, "compute-subnet-1-association", &ec2.RouteTableAssociationArgs{
			SubnetId:     computeSubnet1.ID(),
			RouteTableId: computeRT.ID(),
		})
		if err != nil {
			return err
		}
		_, err = ec2.NewRouteTableAssociation(ctx, "compute-subnet-2-association", &ec2.RouteTableAssociationArgs{
			SubnetId:     computeSubnet2.ID(),
			RouteTableId: computeRT.ID(),
		})
		if err != nil {
			return err
		}
		tgwComputeVPCAttachment, err := ec2transitgateway.NewVpcAttachment(ctx, "compute-vpc-attachment", &ec2transitgateway.VpcAttachmentArgs{
			SubnetIds: pulumi.StringArray{
				computeSubnet1.ID(),
				computeSubnet2.ID(),
			},
			TransitGatewayId: tgw.ID(),
			VpcId:            computeVPC.ID(),
		})
		if err != nil {
			return err
		}

		_, err = ec2transitgateway.NewRouteTableAssociation(ctx, "transit-gateway-route-table-association-compute", &ec2transitgateway.RouteTableAssociationArgs{
			TransitGatewayAttachmentId: tgwComputeVPCAttachment.ID(),
			TransitGatewayRouteTableId: internetTGWRouteTable.ID(),
			ReplaceExistingAssociation: pulumi.Bool(true),
		})
		if err != nil {
			return err
		}
		_, err = ec2transitgateway.NewRouteTablePropagation(ctx, "transit-gateway-route-table-propagation-compute", &ec2transitgateway.RouteTablePropagationArgs{
			TransitGatewayAttachmentId: tgwComputeVPCAttachment.ID(),
			TransitGatewayRouteTableId: internetTGWRouteTable.ID(),
		})
		if err != nil {
			return err
		}

		computeSG, err := ec2.NewSecurityGroup(ctx, "vpc-compute-sg", &ec2.SecurityGroupArgs{
			Ingress: ec2.SecurityGroupIngressArray{
				&ec2.SecurityGroupIngressArgs{
					Description: pulumi.String("traffic from VPC"),
					FromPort:    pulumi.Int(0),
					ToPort:      pulumi.Int(0),
					Protocol:    pulumi.String("-1"),
					CidrBlocks: pulumi.StringArray{
						computeVPC.CidrBlock,
					},
				},
			},
			Egress: ec2.SecurityGroupEgressArray{
				&ec2.SecurityGroupEgressArgs{
					FromPort: pulumi.Int(0),
					ToPort:   pulumi.Int(0),
					Protocol: pulumi.String("-1"),
					CidrBlocks: pulumi.StringArray{
						pulumi.String("0.0.0.0/0"),
					},
				},
			},
			VpcId: computeVPC.ID(),
		})
		if err != nil {
			return err
		}

		computeVPCSsmMessagesEndpoint, err := ec2.NewVpcEndpoint(ctx, "compute-vpc-ssmmessages", &ec2.VpcEndpointArgs{
			PrivateDnsEnabled: pulumi.BoolPtr(true),
			ServiceName:       pulumi.String("com.amazonaws.eu-north-1.ssmmessages"),
			VpcEndpointType:   pulumi.String("Interface"),
			SubnetIds: pulumi.StringArray{
				computeSubnet1.ID(),
				computeSubnet2.ID(),
			},
			SecurityGroupIds: pulumi.StringArray{
				computeSG.ID(),
			},
			VpcId: computeVPC.ID(),
		})
		if err != nil {
			return err
		}

		computeVPCSsmEndpoint, err := ec2.NewVpcEndpoint(ctx, "compute-vpc-ssm", &ec2.VpcEndpointArgs{
			PrivateDnsEnabled: pulumi.BoolPtr(true),
			ServiceName:       pulumi.String("com.amazonaws.eu-north-1.ssm"),
			VpcEndpointType:   pulumi.String("Interface"),
			SubnetIds: pulumi.StringArray{
				computeSubnet1.ID(),
				computeSubnet2.ID(),
			},
			SecurityGroupIds: pulumi.StringArray{
				computeSG.ID(),
			},
			VpcId: computeVPC.ID(),
		})
		if err != nil {
			return err
		}

		computeVPCEC2MessagesEndpoint, err := ec2.NewVpcEndpoint(ctx, "compute-vpc-ec2messages", &ec2.VpcEndpointArgs{
			PrivateDnsEnabled: pulumi.BoolPtr(true),
			ServiceName:       pulumi.String("com.amazonaws.eu-north-1.ec2messages"),
			VpcEndpointType:   pulumi.String("Interface"),
			SubnetIds: pulumi.StringArray{
				computeSubnet1.ID(),
				computeSubnet2.ID(),
			},
			SecurityGroupIds: pulumi.StringArray{
				computeSG.ID(),
			},
			VpcId: computeVPC.ID(),
		})
		if err != nil {
			return err
		}

		_, err = ec2.NewInstance(ctx, "vpcB-EC2", &ec2.InstanceArgs{
			Ami:      pulumi.String("ami-0b5483e9d9802be1f"),
			SubnetId: computeSubnet1.ID(),
			VpcSecurityGroupIds: pulumi.StringArray{
				computeSG.ID(),
			},
			InstanceType:       pulumi.String("t4g.nano"),
			IamInstanceProfile: pulumi.String("ec2-ssm-mgmt"),
		}, pulumi.DependsOn([]pulumi.Resource{computeVPCSsmEndpoint, computeVPCSsmMessagesEndpoint, computeVPCEC2MessagesEndpoint}))

		publicRT, err := ec2.NewRouteTable(ctx, "public-route-table", &ec2.RouteTableArgs{
			Tags: pulumi.StringMap{
				"Name": pulumi.String("egress-vpc-public-rt"),
			},
			VpcId: egressVpc.ID(),
			Routes: ec2.RouteTableRouteArray{
				&ec2.RouteTableRouteArgs{
					CidrBlock: pulumi.String("0.0.0.0/0"),
					GatewayId: igw.ID(),
				},
				&ec2.RouteTableRouteArgs{
					CidrBlock:        computeVPC.CidrBlock,
					TransitGatewayId: tgw.ID(),
				},
			},
		})
		if err != nil {
			return err
		}
		_, err = ec2.NewRouteTableAssociation(ctx, "public-subnet-1-association", &ec2.RouteTableAssociationArgs{
			SubnetId:     publicSubnet1.ID(),
			RouteTableId: publicRT.ID(),
		})
		if err != nil {
			return err
		}
		_, err = ec2.NewRouteTableAssociation(ctx, "public-subnet-2-association", &ec2.RouteTableAssociationArgs{
			SubnetId:     publicSubnet2.ID(),
			RouteTableId: publicRT.ID(),
		})
		if err != nil {
			return err
		}

		return nil
	})
}
