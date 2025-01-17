package aws

import (
	"fmt"
	"log"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/configservice"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	iamwaiter "github.com/terraform-providers/terraform-provider-aws/aws/internal/service/iam/waiter"
)

func resourceAwsConfigDeliveryChannel() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsConfigDeliveryChannelPut,
		Read:   resourceAwsConfigDeliveryChannelRead,
		Update: resourceAwsConfigDeliveryChannelPut,
		Delete: resourceAwsConfigDeliveryChannelDelete,

		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"name": {
				Type:         schema.TypeString,
				Optional:     true,
				ForceNew:     true,
				Default:      "default",
				ValidateFunc: validation.StringLenBetween(0, 256),
			},
			"s3_bucket_name": {
				Type:     schema.TypeString,
				Required: true,
			},
			"s3_key_prefix": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"s3_kms_key_arn": {
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: validateArn,
			},
			"sns_topic_arn": {
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: validateArn,
			},
			"snapshot_delivery_properties": {
				Type:     schema.TypeList,
				Optional: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"delivery_frequency": {
							Type:         schema.TypeString,
							Optional:     true,
							ValidateFunc: validateConfigExecutionFrequency(),
						},
					},
				},
			},
		},
	}
}

func resourceAwsConfigDeliveryChannelPut(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).configconn

	name := d.Get("name").(string)
	channel := configservice.DeliveryChannel{
		Name:         aws.String(name),
		S3BucketName: aws.String(d.Get("s3_bucket_name").(string)),
	}

	if v, ok := d.GetOk("s3_key_prefix"); ok {
		channel.S3KeyPrefix = aws.String(v.(string))
	}
	if v, ok := d.GetOk("s3_kms_key_arn"); ok {
		channel.S3KmsKeyArn = aws.String(v.(string))
	}
	if v, ok := d.GetOk("sns_topic_arn"); ok {
		channel.SnsTopicARN = aws.String(v.(string))
	}

	if p, ok := d.GetOk("snapshot_delivery_properties"); ok {
		propertiesBlocks := p.([]interface{})
		block := propertiesBlocks[0].(map[string]interface{})

		if v, ok := block["delivery_frequency"]; ok {
			channel.ConfigSnapshotDeliveryProperties = &configservice.ConfigSnapshotDeliveryProperties{
				DeliveryFrequency: aws.String(v.(string)),
			}
		}
	}

	input := configservice.PutDeliveryChannelInput{DeliveryChannel: &channel}

	err := resource.Retry(iamwaiter.PropagationTimeout, func() *resource.RetryError {
		_, err := conn.PutDeliveryChannel(&input)
		if err == nil {
			return nil
		}

		if isAWSErr(err, "InsufficientDeliveryPolicyException", "") {
			return resource.RetryableError(err)
		}

		return resource.NonRetryableError(err)
	})
	if isResourceTimeoutError(err) {
		_, err = conn.PutDeliveryChannel(&input)
	}
	if err != nil {
		return fmt.Errorf("Creating Delivery Channel failed: %s", err)
	}

	d.SetId(name)

	return resourceAwsConfigDeliveryChannelRead(d, meta)
}

func resourceAwsConfigDeliveryChannelRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).configconn

	input := configservice.DescribeDeliveryChannelsInput{
		DeliveryChannelNames: []*string{aws.String(d.Id())},
	}
	out, err := conn.DescribeDeliveryChannels(&input)
	if err != nil {
		if awsErr, ok := err.(awserr.Error); ok {
			if awsErr.Code() == "NoSuchDeliveryChannelException" {
				log.Printf("[WARN] Delivery Channel %q is gone (NoSuchDeliveryChannelException)", d.Id())
				d.SetId("")
				return nil
			}
		}
		return fmt.Errorf("Getting Delivery Channel failed: %s", err)
	}

	if len(out.DeliveryChannels) < 1 {
		log.Printf("[WARN] Delivery Channel %q is gone (no channels found)", d.Id())
		d.SetId("")
		return nil
	}

	if len(out.DeliveryChannels) > 1 {
		return fmt.Errorf("Received %d delivery channels under %s (expected exactly 1): %s",
			len(out.DeliveryChannels), d.Id(), out.DeliveryChannels)
	}

	channel := out.DeliveryChannels[0]

	d.Set("name", channel.Name)
	d.Set("s3_bucket_name", channel.S3BucketName)
	d.Set("s3_key_prefix", channel.S3KeyPrefix)
	d.Set("s3_kms_key_arn", channel.S3KmsKeyArn)
	d.Set("sns_topic_arn", channel.SnsTopicARN)

	if channel.ConfigSnapshotDeliveryProperties != nil {
		d.Set("snapshot_delivery_properties", flattenConfigSnapshotDeliveryProperties(channel.ConfigSnapshotDeliveryProperties))
	}

	return nil
}

func resourceAwsConfigDeliveryChannelDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).configconn
	input := configservice.DeleteDeliveryChannelInput{
		DeliveryChannelName: aws.String(d.Id()),
	}

	err := resource.Retry(30*time.Second, func() *resource.RetryError {
		_, err := conn.DeleteDeliveryChannel(&input)
		if err != nil {
			if isAWSErr(err, configservice.ErrCodeLastDeliveryChannelDeleteFailedException, "there is a running configuration recorder") {
				return resource.RetryableError(err)
			}

			return resource.NonRetryableError(err)
		}
		return nil
	})
	if isResourceTimeoutError(err) {
		_, err = conn.DeleteDeliveryChannel(&input)
	}
	if err != nil {
		return fmt.Errorf("Unable to delete delivery channel: %s", err)
	}

	return nil
}
