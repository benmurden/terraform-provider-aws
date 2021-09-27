package ec2_test

import (
	"errors"
	"fmt"
	"log"
	"regexp"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/hashicorp/aws-sdk-go-base/tfawserr"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/hashicorp/terraform-provider-aws/internal/acctest"
	"github.com/hashicorp/terraform-provider-aws/internal/conns"
	tfec2 "github.com/hashicorp/terraform-provider-aws/internal/service/ec2"
	"github.com/hashicorp/terraform-provider-aws/internal/tfresource"
)

func init() {
	resource.AddTestSweepers("aws_ec2_transit_gateway", &resource.Sweeper{
		Name: "aws_ec2_transit_gateway",
		F:    testSweepEc2TransitGateways,
		Dependencies: []string{
			"aws_dx_gateway_association",
			"aws_ec2_transit_gateway_vpc_attachment",
			"aws_ec2_transit_gateway_peering_attachment",
			"aws_vpn_connection",
		},
	})
}

func testSweepEc2TransitGateways(region string) error {
	client, err := acctest.SharedRegionalSweeperClient(region)
	if err != nil {
		return fmt.Errorf("error getting client: %s", err)
	}
	conn := client.(*conns.AWSClient).EC2Conn
	input := &ec2.DescribeTransitGatewaysInput{}

	for {
		output, err := conn.DescribeTransitGateways(input)

		if acctest.SkipSweepError(err) {
			log.Printf("[WARN] Skipping EC2 Transit Gateway sweep for %s: %s", region, err)
			return nil
		}

		if err != nil {
			return fmt.Errorf("error retrieving EC2 Transit Gateways: %s", err)
		}

		for _, transitGateway := range output.TransitGateways {
			if aws.StringValue(transitGateway.State) == ec2.TransitGatewayStateDeleted {
				continue
			}

			id := aws.StringValue(transitGateway.TransitGatewayId)

			input := &ec2.DeleteTransitGatewayInput{
				TransitGatewayId: aws.String(id),
			}

			log.Printf("[INFO] Deleting EC2 Transit Gateway: %s", id)
			err := resource.Retry(2*time.Minute, func() *resource.RetryError {
				_, err := conn.DeleteTransitGateway(input)

				if tfawserr.ErrMessageContains(err, "IncorrectState", "has non-deleted Transit Gateway Attachments") {
					return resource.RetryableError(err)
				}

				if tfawserr.ErrMessageContains(err, "IncorrectState", "has non-deleted DirectConnect Gateway Attachments") {
					return resource.RetryableError(err)
				}

				if tfawserr.ErrMessageContains(err, "IncorrectState", "has non-deleted VPN Attachments") {
					return resource.RetryableError(err)
				}

				if tfawserr.ErrMessageContains(err, "InvalidTransitGatewayID.NotFound", "") {
					return nil
				}

				if err != nil {
					return resource.NonRetryableError(err)
				}

				return nil
			})

			if tfresource.TimedOut(err) {
				_, err = conn.DeleteTransitGateway(input)
			}

			if err != nil {
				return fmt.Errorf("error deleting EC2 Transit Gateway (%s): %s", id, err)
			}

			if err := tfec2.WaitForTransitGatewayDeletion(conn, id); err != nil {
				return fmt.Errorf("error waiting for EC2 Transit Gateway (%s) deletion: %s", id, err)
			}
		}

		if aws.StringValue(output.NextToken) == "" {
			break
		}

		input.NextToken = output.NextToken
	}

	return nil
}

func TestAccEC2TransitGateway_basic(t *testing.T) {
	var transitGateway1 ec2.TransitGateway
	resourceName := "aws_ec2_transit_gateway.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { acctest.PreCheck(t); testAccPreCheckTransitGateway(t) },
		ErrorCheck:   acctest.ErrorCheck(t, ec2.EndpointsID),
		Providers:    acctest.Providers,
		CheckDestroy: testAccCheckTransitGatewayDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccTransitGatewayConfig(),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckTransitGatewayExists(resourceName, &transitGateway1),
					resource.TestCheckResourceAttr(resourceName, "amazon_side_asn", "64512"),
					acctest.MatchResourceAttrRegionalARN(resourceName, "arn", "ec2", regexp.MustCompile(`transit-gateway/tgw-.+`)),
					resource.TestCheckResourceAttrSet(resourceName, "association_default_route_table_id"),
					resource.TestCheckResourceAttr(resourceName, "auto_accept_shared_attachments", ec2.AutoAcceptSharedAttachmentsValueDisable),
					resource.TestCheckResourceAttr(resourceName, "default_route_table_association", ec2.DefaultRouteTableAssociationValueEnable),
					resource.TestCheckResourceAttr(resourceName, "default_route_table_propagation", ec2.DefaultRouteTablePropagationValueEnable),
					resource.TestCheckResourceAttr(resourceName, "description", ""),
					resource.TestCheckResourceAttr(resourceName, "dns_support", ec2.DnsSupportValueEnable),
					acctest.CheckResourceAttrAccountID(resourceName, "owner_id"),
					resource.TestCheckResourceAttrSet(resourceName, "propagation_default_route_table_id"),
					resource.TestCheckResourceAttr(resourceName, "tags.%", "0"),
					resource.TestCheckResourceAttr(resourceName, "vpn_ecmp_support", ec2.VpnEcmpSupportValueEnable),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccEC2TransitGateway_disappears(t *testing.T) {
	var transitGateway1 ec2.TransitGateway
	resourceName := "aws_ec2_transit_gateway.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { acctest.PreCheck(t); testAccPreCheckTransitGateway(t) },
		ErrorCheck:   acctest.ErrorCheck(t, ec2.EndpointsID),
		Providers:    acctest.Providers,
		CheckDestroy: testAccCheckTransitGatewayDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccTransitGatewayConfig(),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckTransitGatewayExists(resourceName, &transitGateway1),
					acctest.CheckResourceDisappears(acctest.Provider, tfec2.ResourceTransitGateway(), resourceName),
				),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func TestAccEC2TransitGateway_amazonSideASN(t *testing.T) {
	var transitGateway1, transitGateway2 ec2.TransitGateway
	resourceName := "aws_ec2_transit_gateway.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { acctest.PreCheck(t); testAccPreCheckTransitGateway(t) },
		ErrorCheck:   acctest.ErrorCheck(t, ec2.EndpointsID),
		Providers:    acctest.Providers,
		CheckDestroy: testAccCheckTransitGatewayDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccTransitGatewayAmazonSideASNConfig(64513),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckTransitGatewayExists(resourceName, &transitGateway1),
					resource.TestCheckResourceAttr(resourceName, "amazon_side_asn", "64513"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: testAccTransitGatewayAmazonSideASNConfig(64514),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckTransitGatewayExists(resourceName, &transitGateway2),
					testAccCheckTransitGatewayRecreated(&transitGateway1, &transitGateway2),
					resource.TestCheckResourceAttr(resourceName, "amazon_side_asn", "64514"),
				),
			},
		},
	})
}

func TestAccEC2TransitGateway_autoAcceptSharedAttachments(t *testing.T) {
	var transitGateway1, transitGateway2 ec2.TransitGateway
	resourceName := "aws_ec2_transit_gateway.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { acctest.PreCheck(t); testAccPreCheckTransitGateway(t) },
		ErrorCheck:   acctest.ErrorCheck(t, ec2.EndpointsID),
		Providers:    acctest.Providers,
		CheckDestroy: testAccCheckTransitGatewayDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccTransitGatewayAutoAcceptSharedAttachmentsConfig(ec2.AutoAcceptSharedAttachmentsValueEnable),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckTransitGatewayExists(resourceName, &transitGateway1),
					resource.TestCheckResourceAttr(resourceName, "auto_accept_shared_attachments", ec2.AutoAcceptSharedAttachmentsValueEnable),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: testAccTransitGatewayAutoAcceptSharedAttachmentsConfig(ec2.AutoAcceptSharedAttachmentsValueDisable),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckTransitGatewayExists(resourceName, &transitGateway2),
					testAccCheckTransitGatewayNotRecreated(&transitGateway1, &transitGateway2),
					resource.TestCheckResourceAttr(resourceName, "auto_accept_shared_attachments", ec2.AutoAcceptSharedAttachmentsValueDisable),
				),
			},
		},
	})
}

func TestAccEC2TransitGateway_defaultRouteTableAssociationAndPropagationDisabled(t *testing.T) {
	var transitGateway1 ec2.TransitGateway
	resourceName := "aws_ec2_transit_gateway.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { acctest.PreCheck(t); testAccPreCheckTransitGateway(t) },
		ErrorCheck:   acctest.ErrorCheck(t, ec2.EndpointsID),
		Providers:    acctest.Providers,
		CheckDestroy: testAccCheckTransitGatewayDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccTransitGatewayDefaultRouteTableAssociationAndPropagationDisabledConfig(),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckTransitGatewayExists(resourceName, &transitGateway1),
					resource.TestCheckResourceAttr(resourceName, "default_route_table_association", ec2.DefaultRouteTableAssociationValueDisable),
					resource.TestCheckResourceAttr(resourceName, "default_route_table_propagation", ec2.DefaultRouteTablePropagationValueDisable),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccEC2TransitGateway_defaultRouteTableAssociation(t *testing.T) {
	var transitGateway1, transitGateway2, transitGateway3 ec2.TransitGateway
	resourceName := "aws_ec2_transit_gateway.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { acctest.PreCheck(t); testAccPreCheckTransitGateway(t) },
		ErrorCheck:   acctest.ErrorCheck(t, ec2.EndpointsID),
		Providers:    acctest.Providers,
		CheckDestroy: testAccCheckTransitGatewayDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccTransitGatewayDefaultRouteTableAssociationConfig(ec2.DefaultRouteTableAssociationValueDisable),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckTransitGatewayExists(resourceName, &transitGateway1),
					resource.TestCheckResourceAttr(resourceName, "default_route_table_association", ec2.DefaultRouteTableAssociationValueDisable),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: testAccTransitGatewayDefaultRouteTableAssociationConfig(ec2.DefaultRouteTableAssociationValueEnable),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckTransitGatewayExists(resourceName, &transitGateway2),
					testAccCheckTransitGatewayRecreated(&transitGateway1, &transitGateway2),
					resource.TestCheckResourceAttr(resourceName, "default_route_table_association", ec2.DefaultRouteTableAssociationValueEnable),
				),
			},
			{
				Config: testAccTransitGatewayDefaultRouteTableAssociationConfig(ec2.DefaultRouteTableAssociationValueDisable),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckTransitGatewayExists(resourceName, &transitGateway3),
					testAccCheckTransitGatewayNotRecreated(&transitGateway2, &transitGateway3),
					resource.TestCheckResourceAttr(resourceName, "default_route_table_association", ec2.DefaultRouteTableAssociationValueDisable),
				),
			},
		},
	})
}

func TestAccEC2TransitGateway_defaultRouteTablePropagation(t *testing.T) {
	var transitGateway1, transitGateway2, transitGateway3 ec2.TransitGateway
	resourceName := "aws_ec2_transit_gateway.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { acctest.PreCheck(t); testAccPreCheckTransitGateway(t) },
		ErrorCheck:   acctest.ErrorCheck(t, ec2.EndpointsID),
		Providers:    acctest.Providers,
		CheckDestroy: testAccCheckTransitGatewayDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccTransitGatewayDefaultRouteTablePropagationConfig(ec2.DefaultRouteTablePropagationValueDisable),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckTransitGatewayExists(resourceName, &transitGateway1),
					resource.TestCheckResourceAttr(resourceName, "default_route_table_propagation", ec2.DefaultRouteTablePropagationValueDisable),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: testAccTransitGatewayDefaultRouteTablePropagationConfig(ec2.DefaultRouteTablePropagationValueEnable),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckTransitGatewayExists(resourceName, &transitGateway2),
					testAccCheckTransitGatewayRecreated(&transitGateway1, &transitGateway2),
					resource.TestCheckResourceAttr(resourceName, "default_route_table_propagation", ec2.DefaultRouteTablePropagationValueEnable),
				),
			},
			{
				Config: testAccTransitGatewayDefaultRouteTablePropagationConfig(ec2.DefaultRouteTablePropagationValueDisable),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckTransitGatewayExists(resourceName, &transitGateway3),
					testAccCheckTransitGatewayNotRecreated(&transitGateway2, &transitGateway3),
					resource.TestCheckResourceAttr(resourceName, "default_route_table_propagation", ec2.DefaultRouteTablePropagationValueDisable),
				),
			},
		},
	})
}

func TestAccEC2TransitGateway_dnsSupport(t *testing.T) {
	var transitGateway1, transitGateway2 ec2.TransitGateway
	resourceName := "aws_ec2_transit_gateway.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { acctest.PreCheck(t); testAccPreCheckTransitGateway(t) },
		ErrorCheck:   acctest.ErrorCheck(t, ec2.EndpointsID),
		Providers:    acctest.Providers,
		CheckDestroy: testAccCheckTransitGatewayDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccTransitGatewayDNSSupportConfig(ec2.DnsSupportValueDisable),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckTransitGatewayExists(resourceName, &transitGateway1),
					resource.TestCheckResourceAttr(resourceName, "dns_support", ec2.DnsSupportValueDisable),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: testAccTransitGatewayDNSSupportConfig(ec2.DnsSupportValueEnable),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckTransitGatewayExists(resourceName, &transitGateway2),
					testAccCheckTransitGatewayNotRecreated(&transitGateway1, &transitGateway2),
					resource.TestCheckResourceAttr(resourceName, "dns_support", ec2.DnsSupportValueEnable),
				),
			},
		},
	})
}

func TestAccEC2TransitGateway_vpnECMPSupport(t *testing.T) {
	var transitGateway1, transitGateway2 ec2.TransitGateway
	resourceName := "aws_ec2_transit_gateway.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { acctest.PreCheck(t); testAccPreCheckTransitGateway(t) },
		ErrorCheck:   acctest.ErrorCheck(t, ec2.EndpointsID),
		Providers:    acctest.Providers,
		CheckDestroy: testAccCheckTransitGatewayDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccTransitGatewayVPNECMPSupportConfig(ec2.VpnEcmpSupportValueDisable),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckTransitGatewayExists(resourceName, &transitGateway1),
					resource.TestCheckResourceAttr(resourceName, "vpn_ecmp_support", ec2.VpnEcmpSupportValueDisable),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: testAccTransitGatewayVPNECMPSupportConfig(ec2.VpnEcmpSupportValueEnable),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckTransitGatewayExists(resourceName, &transitGateway2),
					testAccCheckTransitGatewayNotRecreated(&transitGateway1, &transitGateway2),
					resource.TestCheckResourceAttr(resourceName, "vpn_ecmp_support", ec2.VpnEcmpSupportValueEnable),
				),
			},
		},
	})
}

func TestAccEC2TransitGateway_description(t *testing.T) {
	var transitGateway1, transitGateway2 ec2.TransitGateway
	resourceName := "aws_ec2_transit_gateway.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { acctest.PreCheck(t); testAccPreCheckTransitGateway(t) },
		ErrorCheck:   acctest.ErrorCheck(t, ec2.EndpointsID),
		Providers:    acctest.Providers,
		CheckDestroy: testAccCheckTransitGatewayDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccTransitGatewayDescriptionConfig("description1"),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckTransitGatewayExists(resourceName, &transitGateway1),
					resource.TestCheckResourceAttr(resourceName, "description", "description1"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: testAccTransitGatewayDescriptionConfig("description2"),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckTransitGatewayExists(resourceName, &transitGateway2),
					testAccCheckTransitGatewayNotRecreated(&transitGateway1, &transitGateway2),
					resource.TestCheckResourceAttr(resourceName, "description", "description2"),
				),
			},
		},
	})
}

func TestAccEC2TransitGateway_tags(t *testing.T) {
	var transitGateway1, transitGateway2, transitGateway3 ec2.TransitGateway
	resourceName := "aws_ec2_transit_gateway.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { acctest.PreCheck(t); testAccPreCheckTransitGateway(t) },
		ErrorCheck:   acctest.ErrorCheck(t, ec2.EndpointsID),
		Providers:    acctest.Providers,
		CheckDestroy: testAccCheckTransitGatewayDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccTransitGatewayTags1Config("key1", "value1"),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckTransitGatewayExists(resourceName, &transitGateway1),
					resource.TestCheckResourceAttr(resourceName, "tags.%", "1"),
					resource.TestCheckResourceAttr(resourceName, "tags.key1", "value1"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: testAccTransitGatewayTags2Config("key1", "value1updated", "key2", "value2"),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckTransitGatewayExists(resourceName, &transitGateway2),
					testAccCheckTransitGatewayNotRecreated(&transitGateway1, &transitGateway2),
					resource.TestCheckResourceAttr(resourceName, "tags.%", "2"),
					resource.TestCheckResourceAttr(resourceName, "tags.key1", "value1updated"),
					resource.TestCheckResourceAttr(resourceName, "tags.key2", "value2"),
				),
			},
			{
				Config: testAccTransitGatewayTags1Config("key2", "value2"),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckTransitGatewayExists(resourceName, &transitGateway3),
					testAccCheckTransitGatewayNotRecreated(&transitGateway2, &transitGateway3),
					resource.TestCheckResourceAttr(resourceName, "tags.%", "1"),
					resource.TestCheckResourceAttr(resourceName, "tags.key2", "value2"),
				),
			},
		},
	})
}

func testAccCheckTransitGatewayExists(resourceName string, transitGateway *ec2.TransitGateway) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("Not found: %s", resourceName)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No EC2 Transit Gateway ID is set")
		}

		conn := acctest.Provider.Meta().(*conns.AWSClient).EC2Conn

		gateway, err := tfec2.DescribeTransitGateway(conn, rs.Primary.ID)

		if err != nil {
			return err
		}

		if gateway == nil {
			return fmt.Errorf("EC2 Transit Gateway not found")
		}

		if aws.StringValue(gateway.State) != ec2.TransitGatewayStateAvailable {
			return fmt.Errorf("EC2 Transit Gateway (%s) exists in non-available (%s) state", rs.Primary.ID, aws.StringValue(gateway.State))
		}

		*transitGateway = *gateway

		return nil
	}
}

func testAccCheckTransitGatewayDestroy(s *terraform.State) error {
	conn := acctest.Provider.Meta().(*conns.AWSClient).EC2Conn

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "aws_ec2_transit_gateway" {
			continue
		}

		transitGateway, err := tfec2.DescribeTransitGateway(conn, rs.Primary.ID)

		if tfawserr.ErrMessageContains(err, "InvalidTransitGatewayID.NotFound", "") {
			continue
		}

		if err != nil {
			return err
		}

		if transitGateway == nil {
			continue
		}

		if aws.StringValue(transitGateway.State) != ec2.TransitGatewayStateDeleted {
			return fmt.Errorf("EC2 Transit Gateway (%s) still exists in non-deleted (%s) state", rs.Primary.ID, aws.StringValue(transitGateway.State))
		}
	}

	return nil
}

func testAccCheckTransitGatewayNotRecreated(i, j *ec2.TransitGateway) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if aws.StringValue(i.TransitGatewayId) != aws.StringValue(j.TransitGatewayId) {
			return errors.New("EC2 Transit Gateway was recreated")
		}

		return nil
	}
}

func testAccCheckTransitGatewayRecreated(i, j *ec2.TransitGateway) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if aws.StringValue(i.TransitGatewayId) == aws.StringValue(j.TransitGatewayId) {
			return errors.New("EC2 Transit Gateway was not recreated")
		}

		return nil
	}
}

func testAccCheckTransitGatewayAssociationDefaultRouteTableVPCAttachmentAssociated(transitGateway *ec2.TransitGateway, transitGatewayVpcAttachment *ec2.TransitGatewayVpcAttachment) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		conn := acctest.Provider.Meta().(*conns.AWSClient).EC2Conn

		attachmentID := aws.StringValue(transitGatewayVpcAttachment.TransitGatewayAttachmentId)
		routeTableID := aws.StringValue(transitGateway.Options.AssociationDefaultRouteTableId)
		association, err := tfec2.DescribeTransitGatewayRouteTableAssociation(conn, routeTableID, attachmentID)

		if err != nil {
			return err
		}

		if association == nil {
			return errors.New("EC2 Transit Gateway Route Table Association not found")
		}

		return nil
	}
}

func testAccCheckTransitGatewayAssociationDefaultRouteTableVPCAttachmentNotAssociated(transitGateway *ec2.TransitGateway, transitGatewayVpcAttachment *ec2.TransitGatewayVpcAttachment) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		conn := acctest.Provider.Meta().(*conns.AWSClient).EC2Conn

		attachmentID := aws.StringValue(transitGatewayVpcAttachment.TransitGatewayAttachmentId)
		routeTableID := aws.StringValue(transitGateway.Options.AssociationDefaultRouteTableId)
		association, err := tfec2.DescribeTransitGatewayRouteTableAssociation(conn, routeTableID, attachmentID)

		if err != nil {
			return err
		}

		if association != nil {
			return errors.New("EC2 Transit Gateway Route Table Association found")
		}

		return nil
	}
}

func testAccCheckTransitGatewayPropagationDefaultRouteTableVPCAttachmentNotPropagated(transitGateway *ec2.TransitGateway, transitGatewayVpcAttachment *ec2.TransitGatewayVpcAttachment) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		conn := acctest.Provider.Meta().(*conns.AWSClient).EC2Conn

		attachmentID := aws.StringValue(transitGatewayVpcAttachment.TransitGatewayAttachmentId)
		routeTableID := aws.StringValue(transitGateway.Options.AssociationDefaultRouteTableId)
		propagation, err := tfec2.FindTransitGatewayRouteTablePropagation(conn, routeTableID, attachmentID)

		if err != nil {
			return err
		}

		if propagation != nil {
			return errors.New("EC2 Transit Gateway Route Table Propagation enabled")
		}

		return nil
	}
}

func testAccCheckTransitGatewayPropagationDefaultRouteTableVPCAttachmentPropagated(transitGateway *ec2.TransitGateway, transitGatewayVpcAttachment *ec2.TransitGatewayVpcAttachment) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		conn := acctest.Provider.Meta().(*conns.AWSClient).EC2Conn

		attachmentID := aws.StringValue(transitGatewayVpcAttachment.TransitGatewayAttachmentId)
		routeTableID := aws.StringValue(transitGateway.Options.AssociationDefaultRouteTableId)
		propagation, err := tfec2.FindTransitGatewayRouteTablePropagation(conn, routeTableID, attachmentID)

		if err != nil {
			return err
		}

		if propagation == nil {
			return errors.New("EC2 Transit Gateway Route Table Propagation not enabled")
		}

		return nil
	}
}

func testAccPreCheckTransitGateway(t *testing.T) {
	conn := acctest.Provider.Meta().(*conns.AWSClient).EC2Conn

	input := &ec2.DescribeTransitGatewaysInput{
		MaxResults: aws.Int64(5),
	}

	_, err := conn.DescribeTransitGateways(input)

	if acctest.PreCheckSkipError(err) || tfawserr.ErrMessageContains(err, "InvalidAction", "") {
		t.Skipf("skipping acceptance testing: %s", err)
	}

	if err != nil {
		t.Fatalf("unexpected PreCheck error: %s", err)
	}
}

func testAccTransitGatewayConfig() string {
	return `
resource "aws_ec2_transit_gateway" "test" {}
`
}

func testAccTransitGatewayAmazonSideASNConfig(amazonSideASN int) string {
	return fmt.Sprintf(`
resource "aws_ec2_transit_gateway" "test" {
  amazon_side_asn = %d
}
`, amazonSideASN)
}

func testAccTransitGatewayAutoAcceptSharedAttachmentsConfig(autoAcceptSharedAttachments string) string {
	return fmt.Sprintf(`
resource "aws_ec2_transit_gateway" "test" {
  auto_accept_shared_attachments = %q
}
`, autoAcceptSharedAttachments)
}

func testAccTransitGatewayDefaultRouteTableAssociationAndPropagationDisabledConfig() string {
	return `
resource "aws_ec2_transit_gateway" "test" {
  default_route_table_association = "disable"
  default_route_table_propagation = "disable"
}
`
}

func testAccTransitGatewayDefaultRouteTableAssociationConfig(defaultRouteTableAssociation string) string {
	return fmt.Sprintf(`
resource "aws_ec2_transit_gateway" "test" {
  default_route_table_association = %q
}
`, defaultRouteTableAssociation)
}

func testAccTransitGatewayDefaultRouteTablePropagationConfig(defaultRouteTablePropagation string) string {
	return fmt.Sprintf(`
resource "aws_ec2_transit_gateway" "test" {
  default_route_table_propagation = %q
}
`, defaultRouteTablePropagation)
}

func testAccTransitGatewayDNSSupportConfig(dnsSupport string) string {
	return fmt.Sprintf(`
resource "aws_ec2_transit_gateway" "test" {
  dns_support = %q
}
`, dnsSupport)
}

func testAccTransitGatewayVPNECMPSupportConfig(vpnEcmpSupport string) string {
	return fmt.Sprintf(`
resource "aws_ec2_transit_gateway" "test" {
  vpn_ecmp_support = %q
}
`, vpnEcmpSupport)
}

func testAccTransitGatewayDescriptionConfig(description string) string {
	return fmt.Sprintf(`
resource "aws_ec2_transit_gateway" "test" {
  description = %q
}
`, description)
}

func testAccTransitGatewayTags1Config(tagKey1, tagValue1 string) string {
	return fmt.Sprintf(`
resource "aws_ec2_transit_gateway" "test" {
  tags = {
    %q = %q
  }
}
`, tagKey1, tagValue1)
}

func testAccTransitGatewayTags2Config(tagKey1, tagValue1, tagKey2, tagValue2 string) string {
	return fmt.Sprintf(`
resource "aws_ec2_transit_gateway" "test" {
  tags = {
    %q = %q
    %q = %q
  }
}
`, tagKey1, tagValue1, tagKey2, tagValue2)
}
