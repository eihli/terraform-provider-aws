package aws

import (
	"errors"
	"fmt"
	"log"
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/cognitoidentityprovider"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func init() {
	RegisterServiceErrorCheckFunc(cognitoidentityprovider.EndpointsID, testAccErrorCheckSkipCognitoIdentityProvider)

	resource.AddTestSweepers("aws_cognito_user_pool", &resource.Sweeper{
		Name: "aws_cognito_user_pool",
		F:    testSweepCognitoUserPools,
		Dependencies: []string{
			"aws_cognito_user_pool_domain",
		},
	})
}

func testSweepCognitoUserPools(region string) error {
	client, err := sharedClientForRegion(region)
	if err != nil {
		return fmt.Errorf("error getting client: %w", err)
	}
	conn := client.(*AWSClient).cognitoidpconn

	input := &cognitoidentityprovider.ListUserPoolsInput{
		MaxResults: aws.Int64(int64(50)),
	}

	err = conn.ListUserPoolsPages(input, func(resp *cognitoidentityprovider.ListUserPoolsOutput, lastPage bool) bool {
		if len(resp.UserPools) == 0 {
			log.Print("[DEBUG] No Cognito User Pools to sweep")
			return false
		}

		for _, userPool := range resp.UserPools {
			name := aws.StringValue(userPool.Name)

			log.Printf("[INFO] Deleting Cognito User Pool: %s", name)
			_, err := conn.DeleteUserPool(&cognitoidentityprovider.DeleteUserPoolInput{
				UserPoolId: userPool.Id,
			})
			if err != nil {
				log.Printf("[ERROR] Failed deleting Cognito User Pool (%s): %s", name, err)
			}
		}
		return !lastPage
	})

	if err != nil {
		if testSweepSkipSweepError(err) {
			log.Printf("[WARN] Skipping Cognito User Pool sweep for %s: %s", region, err)
			return nil
		}
		return fmt.Errorf("Error retrieving Cognito User Pools: %w", err)
	}

	return nil
}

func testAccErrorCheckSkipCognitoIdentityProvider(t *testing.T) resource.ErrorCheckFunc {
	return testAccErrorCheckSkipMessagesContaining(t,
		"not supported in this region",
	)
}

func TestAccAWSCognitoUserPool_basic(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc-test")
	resourceName := "aws_cognito_user_pool.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t); testAccPreCheckAWSCognitoIdentityProvider(t) },
		ErrorCheck:   testAccErrorCheck(t, cognitoidentityprovider.EndpointsID),
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSCognitoUserPoolDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSCognitoUserPoolConfig_Name(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckAWSCognitoUserPoolExists(resourceName, nil),
					testAccMatchResourceAttrRegionalARN(resourceName, "arn", "cognito-idp", regexp.MustCompile(`userpool/.+`)),
					resource.TestMatchResourceAttr(resourceName, "endpoint", regexp.MustCompile(`^cognito-idp\.[^.]+\.amazonaws.com/[\w-]+_[0-9a-zA-Z]+$`)),
					resource.TestCheckResourceAttr(resourceName, "name", rName),
					resource.TestCheckResourceAttrSet(resourceName, "creation_date"),
					resource.TestCheckResourceAttrSet(resourceName, "last_modified_date"),
					resource.TestCheckResourceAttr(resourceName, "mfa_configuration", "OFF"),
					resource.TestCheckResourceAttr(resourceName, "sms_configuration.#", "0"),
					resource.TestCheckResourceAttr(resourceName, "software_token_mfa_configuration.#", "0"),
					resource.TestCheckResourceAttr(resourceName, "account_recovery_setting.#", "0"),
					resource.TestCheckResourceAttr(resourceName, "admin_create_user_config.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "admin_create_user_config.0.allow_admin_create_user_only", "false"),
					resource.TestCheckResourceAttr(resourceName, "email_configuration.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "email_configuration.0.email_sending_account", "COGNITO_DEFAULT"),
					resource.TestCheckResourceAttr(resourceName, "password_policy.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "password_policy.0.minimum_length", "8"),
					resource.TestCheckResourceAttr(resourceName, "password_policy.0.require_lowercase", "true"),
					resource.TestCheckResourceAttr(resourceName, "password_policy.0.require_numbers", "true"),
					resource.TestCheckResourceAttr(resourceName, "password_policy.0.require_symbols", "true"),
					resource.TestCheckResourceAttr(resourceName, "password_policy.0.require_uppercase", "true"),
					resource.TestCheckResourceAttr(resourceName, "password_policy.0.temporary_password_validity_days", "7"),
					resource.TestCheckResourceAttr(resourceName, "verification_message_template.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "verification_message_template.0.default_email_option", "CONFIRM_WITH_CODE"),
					resource.TestCheckResourceAttr(resourceName, "lambda_config.#", "0"),
					resource.TestCheckResourceAttr(resourceName, "tags.%", "0"),
					resource.TestCheckResourceAttr(resourceName, "estimated_number_of_users", "0"),
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

func TestAccAWSCognitoUserPool_recovery(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc-test")
	resourceName := "aws_cognito_user_pool.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t); testAccPreCheckAWSCognitoIdentityProvider(t) },
		ErrorCheck:   testAccErrorCheck(t, cognitoidentityprovider.EndpointsID),
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSCognitoUserPoolDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSCognitoUserPoolConfigAccountRecoverySingle(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckAWSCognitoUserPoolExists(resourceName, nil),
					resource.TestCheckResourceAttr(resourceName, "name", rName),
					resource.TestCheckResourceAttr(resourceName, "account_recovery_setting.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "account_recovery_setting.0.recovery_mechanism.#", "1"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: testAccAWSCognitoUserPoolConfigAccountRecoveryMulti(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckAWSCognitoUserPoolExists(resourceName, nil),
					resource.TestCheckResourceAttr(resourceName, "name", rName),
					resource.TestCheckResourceAttr(resourceName, "account_recovery_setting.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "account_recovery_setting.0.recovery_mechanism.#", "2"),
				),
			},
			{
				Config: testAccAWSCognitoUserPoolConfigAccountRecoveryUpdate(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckAWSCognitoUserPoolExists(resourceName, nil),
					resource.TestCheckResourceAttr(resourceName, "name", rName),
					resource.TestCheckResourceAttr(resourceName, "account_recovery_setting.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "account_recovery_setting.0.recovery_mechanism.#", "1"),
				),
			},
		},
	})
}

func TestAccAWSCognitoUserPool_withAdminCreateUserConfiguration(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc-test")
	resourceName := "aws_cognito_user_pool.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t); testAccPreCheckAWSCognitoIdentityProvider(t) },
		ErrorCheck:   testAccErrorCheck(t, cognitoidentityprovider.EndpointsID),
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSCognitoUserPoolDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSCognitoUserPoolConfig_withAdminCreateUserConfiguration(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckAWSCognitoUserPoolExists(resourceName, nil),
					resource.TestCheckResourceAttr(resourceName, "admin_create_user_config.0.allow_admin_create_user_only", "true"),
					resource.TestCheckResourceAttr(resourceName, "admin_create_user_config.0.invite_message_template.0.email_message", "Your username is {username} and temporary password is {####}. "),
					resource.TestCheckResourceAttr(resourceName, "admin_create_user_config.0.invite_message_template.0.email_subject", "FooBar {####}"),
					resource.TestCheckResourceAttr(resourceName, "admin_create_user_config.0.invite_message_template.0.sms_message", "Your username is {username} and temporary password is {####}."),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: testAccAWSCognitoUserPoolConfig_withAdminCreateUserConfigurationUpdated(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "admin_create_user_config.0.allow_admin_create_user_only", "false"),
					resource.TestCheckResourceAttr(resourceName, "admin_create_user_config.0.invite_message_template.0.email_message", "Your username is {username} and constant password is {####}. "),
					resource.TestCheckResourceAttr(resourceName, "admin_create_user_config.0.invite_message_template.0.email_subject", "Foo{####}BaBaz"),
					resource.TestCheckResourceAttr(resourceName, "admin_create_user_config.0.invite_message_template.0.sms_message", "Your username is {username} and constant password is {####}."),
				),
			},
		},
	})
}

// Reference: https://github.com/hashicorp/terraform-provider-aws/issues/11858
func TestAccAWSCognitoUserPool_withAdminCreateUserConfigurationAndPasswordPolicy(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc-test")
	resourceName := "aws_cognito_user_pool.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t); testAccPreCheckAWSCognitoIdentityProvider(t) },
		ErrorCheck:   testAccErrorCheck(t, cognitoidentityprovider.EndpointsID),
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSCognitoUserPoolDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSCognitoUserPoolConfig_withAdminCreateUserConfigAndPasswordPolicy(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckAWSCognitoUserPoolExists(resourceName, nil),
					resource.TestCheckResourceAttr(resourceName, "admin_create_user_config.0.allow_admin_create_user_only", "true"),
					resource.TestCheckResourceAttr(resourceName, "password_policy.0.temporary_password_validity_days", "7"),
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

func TestAccAWSCognitoUserPool_withAdvancedSecurityMode(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc-test")
	resourceName := "aws_cognito_user_pool.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t); testAccPreCheckAWSCognitoIdentityProvider(t) },
		ErrorCheck:   testAccErrorCheck(t, cognitoidentityprovider.EndpointsID),
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSCognitoUserPoolDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSCognitoUserPoolConfig_AdvancedSecurityMode(rName, "OFF"),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckAWSCognitoUserPoolExists(resourceName, nil),
					resource.TestCheckResourceAttr(resourceName, "user_pool_add_ons.0.advanced_security_mode", "OFF"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: testAccAWSCognitoUserPoolConfig_AdvancedSecurityMode(rName, "ENFORCED"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "user_pool_add_ons.0.advanced_security_mode", "ENFORCED"),
				),
			},
			{
				Config: testAccAWSCognitoUserPoolConfig_Name(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "user_pool_add_ons.#", "0"),
				),
			},
		},
	})
}

func TestAccAWSCognitoUserPool_withDeviceConfiguration(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc-test")
	resourceName := "aws_cognito_user_pool.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t); testAccPreCheckAWSCognitoIdentityProvider(t) },
		ErrorCheck:   testAccErrorCheck(t, cognitoidentityprovider.EndpointsID),
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSCognitoUserPoolDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSCognitoUserPoolConfig_withDeviceConfiguration(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckAWSCognitoUserPoolExists(resourceName, nil),
					resource.TestCheckResourceAttr(resourceName, "device_configuration.0.challenge_required_on_new_device", "true"),
					resource.TestCheckResourceAttr(resourceName, "device_configuration.0.device_only_remembered_on_user_prompt", "false"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: testAccAWSCognitoUserPoolConfig_withDeviceConfigurationUpdated(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "device_configuration.0.challenge_required_on_new_device", "false"),
					resource.TestCheckResourceAttr(resourceName, "device_configuration.0.device_only_remembered_on_user_prompt", "true"),
				),
			},
		},
	})
}

func TestAccAWSCognitoUserPool_withEmailVerificationMessage(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc-test")
	subject := acctest.RandString(10)
	updatedSubject := acctest.RandString(10)
	message := fmt.Sprintf("%s {####}", acctest.RandString(10))
	upatedMessage := fmt.Sprintf("%s {####}", acctest.RandString(10))
	resourceName := "aws_cognito_user_pool.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t); testAccPreCheckAWSCognitoIdentityProvider(t) },
		ErrorCheck:   testAccErrorCheck(t, cognitoidentityprovider.EndpointsID),
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSCognitoUserPoolDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSCognitoUserPoolConfig_withEmailVerificationMessage(rName, subject, message),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckAWSCognitoUserPoolExists(resourceName, nil),
					resource.TestCheckResourceAttr(resourceName, "email_verification_subject", subject),
					resource.TestCheckResourceAttr(resourceName, "email_verification_message", message),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: testAccAWSCognitoUserPoolConfig_withEmailVerificationMessage(rName, updatedSubject, upatedMessage),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "email_verification_subject", updatedSubject),
					resource.TestCheckResourceAttr(resourceName, "email_verification_message", upatedMessage),
				),
			},
		},
	})
}

func TestAccAWSCognitoUserPool_MfaConfiguration_SmsConfiguration(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc-test")
	iamRoleResourceName := "aws_iam_role.test"
	resourceName := "aws_cognito_user_pool.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t); testAccPreCheckAWSCognitoIdentityProvider(t) },
		ErrorCheck:   testAccErrorCheck(t, cognitoidentityprovider.EndpointsID),
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSCognitoUserPoolDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSCognitoUserPoolConfig_MfaConfiguration_SmsConfiguration(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "mfa_configuration", "ON"),
					resource.TestCheckResourceAttr(resourceName, "sms_configuration.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "sms_configuration.0.external_id", "test"),
					resource.TestCheckResourceAttrPair(resourceName, "sms_configuration.0.sns_caller_arn", iamRoleResourceName, "arn"),
					resource.TestCheckResourceAttr(resourceName, "software_token_mfa_configuration.#", "0"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: testAccAWSCognitoUserPoolConfig_MfaConfiguration(rName, cognitoidentityprovider.UserPoolMfaTypeOff),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "mfa_configuration", "OFF"),
					resource.TestCheckResourceAttr(resourceName, "sms_configuration.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "software_token_mfa_configuration.#", "0"),
				),
			},
			{
				Config: testAccAWSCognitoUserPoolConfig_MfaConfiguration_SmsConfiguration(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "mfa_configuration", "ON"),
					resource.TestCheckResourceAttr(resourceName, "sms_configuration.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "sms_configuration.0.external_id", "test"),
					resource.TestCheckResourceAttrPair(resourceName, "sms_configuration.0.sns_caller_arn", iamRoleResourceName, "arn"),
					resource.TestCheckResourceAttr(resourceName, "software_token_mfa_configuration.#", "0"),
				),
			},
		},
	})
}

func TestAccAWSCognitoUserPool_MfaConfiguration_SmsConfigurationAndSoftwareTokenMfaConfiguration(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc-test")
	iamRoleResourceName := "aws_iam_role.test"
	resourceName := "aws_cognito_user_pool.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t); testAccPreCheckAWSCognitoIdentityProvider(t) },
		ErrorCheck:   testAccErrorCheck(t, cognitoidentityprovider.EndpointsID),
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSCognitoUserPoolDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSCognitoUserPoolConfig_MfaConfiguration_SmsConfigurationAndSoftwareTokenMfaConfigurationEnabled(rName, false),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "mfa_configuration", "ON"),
					resource.TestCheckResourceAttr(resourceName, "sms_configuration.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "sms_configuration.0.external_id", "test"),
					resource.TestCheckResourceAttrPair(resourceName, "sms_configuration.0.sns_caller_arn", iamRoleResourceName, "arn"),
					resource.TestCheckResourceAttr(resourceName, "software_token_mfa_configuration.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "software_token_mfa_configuration.0.enabled", "false"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: testAccAWSCognitoUserPoolConfig_MfaConfiguration_SmsConfigurationAndSoftwareTokenMfaConfigurationEnabled(rName, true),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "mfa_configuration", "ON"),
					resource.TestCheckResourceAttr(resourceName, "sms_configuration.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "sms_configuration.0.external_id", "test"),
					resource.TestCheckResourceAttrPair(resourceName, "sms_configuration.0.sns_caller_arn", iamRoleResourceName, "arn"),
					resource.TestCheckResourceAttr(resourceName, "software_token_mfa_configuration.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "software_token_mfa_configuration.0.enabled", "true"),
				),
			},
			{
				Config: testAccAWSCognitoUserPoolConfig_MfaConfiguration(rName, cognitoidentityprovider.UserPoolMfaTypeOff),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "mfa_configuration", "OFF"),
					resource.TestCheckResourceAttr(resourceName, "sms_configuration.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "software_token_mfa_configuration.#", "0"),
				),
			},
		},
	})
}

func TestAccAWSCognitoUserPool_MfaConfiguration_SmsConfigurationToSoftwareTokenMfaConfiguration(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc-test")
	iamRoleResourceName := "aws_iam_role.test"
	resourceName := "aws_cognito_user_pool.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t); testAccPreCheckAWSCognitoIdentityProvider(t) },
		ErrorCheck:   testAccErrorCheck(t, cognitoidentityprovider.EndpointsID),
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSCognitoUserPoolDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSCognitoUserPoolConfig_MfaConfiguration_SmsConfiguration(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "mfa_configuration", "ON"),
					resource.TestCheckResourceAttr(resourceName, "sms_configuration.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "sms_configuration.0.external_id", "test"),
					resource.TestCheckResourceAttrPair(resourceName, "sms_configuration.0.sns_caller_arn", iamRoleResourceName, "arn"),
					resource.TestCheckResourceAttr(resourceName, "software_token_mfa_configuration.#", "0"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: testAccAWSCognitoUserPoolConfig_MfaConfiguration_SoftwareTokenMfaConfigurationEnabled(rName, true),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "mfa_configuration", "ON"),
					resource.TestCheckResourceAttr(resourceName, "sms_configuration.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "software_token_mfa_configuration.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "software_token_mfa_configuration.0.enabled", "true"),
				),
			},
		},
	})
}

func TestAccAWSCognitoUserPool_MfaConfiguration_SoftwareTokenMfaConfiguration(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc-test")
	resourceName := "aws_cognito_user_pool.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t); testAccPreCheckAWSCognitoIdentityProvider(t) },
		ErrorCheck:   testAccErrorCheck(t, cognitoidentityprovider.EndpointsID),
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSCognitoUserPoolDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSCognitoUserPoolConfig_MfaConfiguration_SoftwareTokenMfaConfigurationEnabled(rName, true),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "mfa_configuration", "ON"),
					resource.TestCheckResourceAttr(resourceName, "sms_configuration.#", "0"),
					resource.TestCheckResourceAttr(resourceName, "software_token_mfa_configuration.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "software_token_mfa_configuration.0.enabled", "true"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: testAccAWSCognitoUserPoolConfig_MfaConfiguration(rName, cognitoidentityprovider.UserPoolMfaTypeOff),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "mfa_configuration", "OFF"),
					resource.TestCheckResourceAttr(resourceName, "sms_configuration.#", "0"),
					resource.TestCheckResourceAttr(resourceName, "software_token_mfa_configuration.#", "0"),
				),
			},
			{
				Config: testAccAWSCognitoUserPoolConfig_MfaConfiguration_SoftwareTokenMfaConfigurationEnabled(rName, true),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "mfa_configuration", "ON"),
					resource.TestCheckResourceAttr(resourceName, "sms_configuration.#", "0"),
					resource.TestCheckResourceAttr(resourceName, "software_token_mfa_configuration.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "software_token_mfa_configuration.0.enabled", "true"),
				),
			},
		},
	})
}

func TestAccAWSCognitoUserPool_MfaConfiguration_SoftwareTokenMfaConfigurationToSmsConfiguration(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc-test")
	iamRoleResourceName := "aws_iam_role.test"
	resourceName := "aws_cognito_user_pool.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t); testAccPreCheckAWSCognitoIdentityProvider(t) },
		ErrorCheck:   testAccErrorCheck(t, cognitoidentityprovider.EndpointsID),
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSCognitoUserPoolDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSCognitoUserPoolConfig_MfaConfiguration_SoftwareTokenMfaConfigurationEnabled(rName, true),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "mfa_configuration", "ON"),
					resource.TestCheckResourceAttr(resourceName, "sms_configuration.#", "0"),
					resource.TestCheckResourceAttr(resourceName, "software_token_mfa_configuration.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "software_token_mfa_configuration.0.enabled", "true"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: testAccAWSCognitoUserPoolConfig_MfaConfiguration_SmsConfiguration(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "mfa_configuration", "ON"),
					resource.TestCheckResourceAttr(resourceName, "sms_configuration.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "sms_configuration.0.external_id", "test"),
					resource.TestCheckResourceAttrPair(resourceName, "sms_configuration.0.sns_caller_arn", iamRoleResourceName, "arn"),
					resource.TestCheckResourceAttr(resourceName, "software_token_mfa_configuration.#", "0"),
				),
			},
		},
	})
}

func TestAccAWSCognitoUserPool_SmsAuthenticationMessage(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc-test")
	smsAuthenticationMessage1 := "test authentication message {####}"
	smsAuthenticationMessage2 := "test authentication message updated {####}"
	resourceName := "aws_cognito_user_pool.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t); testAccPreCheckAWSCognitoIdentityProvider(t) },
		ErrorCheck:   testAccErrorCheck(t, cognitoidentityprovider.EndpointsID),
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSCognitoUserPoolDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSCognitoUserPoolConfig_SmsAuthenticationMessage(rName, smsAuthenticationMessage1),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckAWSCognitoUserPoolExists(resourceName, nil),
					resource.TestCheckResourceAttr(resourceName, "sms_authentication_message", smsAuthenticationMessage1),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: testAccAWSCognitoUserPoolConfig_SmsAuthenticationMessage(rName, smsAuthenticationMessage2),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckAWSCognitoUserPoolExists(resourceName, nil),
					resource.TestCheckResourceAttr(resourceName, "sms_authentication_message", smsAuthenticationMessage2),
				),
			},
		},
	})
}

func TestAccAWSCognitoUserPool_SmsConfiguration(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc-test")
	iamRoleResourceName := "aws_iam_role.test"
	resourceName := "aws_cognito_user_pool.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t); testAccPreCheckAWSCognitoIdentityProvider(t) },
		ErrorCheck:   testAccErrorCheck(t, cognitoidentityprovider.EndpointsID),
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSCognitoUserPoolDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSCognitoUserPoolConfig_SmsConfiguration_ExternalId(rName, "test"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "mfa_configuration", "OFF"),
					resource.TestCheckResourceAttr(resourceName, "sms_configuration.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "sms_configuration.0.external_id", "test"),
					resource.TestCheckResourceAttrPair(resourceName, "sms_configuration.0.sns_caller_arn", iamRoleResourceName, "arn"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: testAccAWSCognitoUserPoolConfig_Name(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "mfa_configuration", "OFF"),
					resource.TestCheckResourceAttr(resourceName, "sms_configuration.#", "1"),
				),
			},
			{
				Config: testAccAWSCognitoUserPoolConfig_SmsConfiguration_ExternalId(rName, "test"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "mfa_configuration", "OFF"),
					resource.TestCheckResourceAttr(resourceName, "sms_configuration.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "sms_configuration.0.external_id", "test"),
					resource.TestCheckResourceAttrPair(resourceName, "sms_configuration.0.sns_caller_arn", iamRoleResourceName, "arn"),
				),
			},
		},
	})
}

func TestAccAWSCognitoUserPool_SmsConfiguration_ExternalId(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc-test")
	iamRoleResourceName := "aws_iam_role.test"
	resourceName := "aws_cognito_user_pool.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t); testAccPreCheckAWSCognitoIdentityProvider(t) },
		ErrorCheck:   testAccErrorCheck(t, cognitoidentityprovider.EndpointsID),
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSCognitoUserPoolDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSCognitoUserPoolConfig_SmsConfiguration_ExternalId(rName, "test"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "mfa_configuration", "OFF"),
					resource.TestCheckResourceAttr(resourceName, "sms_configuration.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "sms_configuration.0.external_id", "test"),
					resource.TestCheckResourceAttrPair(resourceName, "sms_configuration.0.sns_caller_arn", iamRoleResourceName, "arn"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: testAccAWSCognitoUserPoolConfig_SmsConfiguration_ExternalId(rName, "test2"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "mfa_configuration", "OFF"),
					resource.TestCheckResourceAttr(resourceName, "sms_configuration.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "sms_configuration.0.external_id", "test2"),
					resource.TestCheckResourceAttrPair(resourceName, "sms_configuration.0.sns_caller_arn", iamRoleResourceName, "arn"),
				),
			},
		},
	})
}

func TestAccAWSCognitoUserPool_SmsConfiguration_SnsCallerArn(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc-test")
	iamRoleResourceName := "aws_iam_role.test"
	resourceName := "aws_cognito_user_pool.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t); testAccPreCheckAWSCognitoIdentityProvider(t) },
		ErrorCheck:   testAccErrorCheck(t, cognitoidentityprovider.EndpointsID),
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSCognitoUserPoolDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSCognitoUserPoolConfig_SmsConfiguration_ExternalId(rName, "test"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "mfa_configuration", "OFF"),
					resource.TestCheckResourceAttr(resourceName, "sms_configuration.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "sms_configuration.0.external_id", "test"),
					resource.TestCheckResourceAttrPair(resourceName, "sms_configuration.0.sns_caller_arn", iamRoleResourceName, "arn"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: testAccAWSCognitoUserPoolConfig_SmsConfiguration_SnsCallerArn2(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "mfa_configuration", "OFF"),
					resource.TestCheckResourceAttr(resourceName, "sms_configuration.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "sms_configuration.0.external_id", "test"),
					resource.TestCheckResourceAttrPair(resourceName, "sms_configuration.0.sns_caller_arn", iamRoleResourceName, "arn"),
				),
			},
		},
	})
}

func TestAccAWSCognitoUserPool_SmsVerificationMessage(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc-test")
	smsVerificationMessage1 := "test verification message {####}"
	smsVerificationMessage2 := "test verification message updated {####}"
	resourceName := "aws_cognito_user_pool.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t); testAccPreCheckAWSCognitoIdentityProvider(t) },
		ErrorCheck:   testAccErrorCheck(t, cognitoidentityprovider.EndpointsID),
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSCognitoUserPoolDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSCognitoUserPoolConfig_SmsVerificationMessage(rName, smsVerificationMessage1),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckAWSCognitoUserPoolExists(resourceName, nil),
					resource.TestCheckResourceAttr(resourceName, "sms_verification_message", smsVerificationMessage1),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: testAccAWSCognitoUserPoolConfig_SmsVerificationMessage(rName, smsVerificationMessage2),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckAWSCognitoUserPoolExists(resourceName, nil),
					resource.TestCheckResourceAttr(resourceName, "sms_verification_message", smsVerificationMessage2),
				),
			},
		},
	})
}

func TestAccAWSCognitoUserPool_withEmailConfiguration(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc-test")
	resourceName := "aws_cognito_user_pool.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t); testAccPreCheckAWSCognitoIdentityProvider(t) },
		ErrorCheck:   testAccErrorCheck(t, cognitoidentityprovider.EndpointsID),
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSCognitoUserPoolDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSCognitoUserPoolConfig_withEmailConfiguration(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "email_configuration.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "email_configuration.0.reply_to_email_address", ""),
					resource.TestCheckResourceAttr(resourceName, "email_configuration.0.email_sending_account", "COGNITO_DEFAULT"),
					resource.TestCheckResourceAttr(resourceName, "email_configuration.0.from_email_address", ""),
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

func TestAccAWSCognitoUserPool_withEmailConfigurationSource(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc-test")
	replyTo := testAccDefaultEmailAddress
	resourceName := "aws_cognito_user_pool.test"
	resourceName2 := "aws_ses_configuration_set.test"

	sourceARN, ok := os.LookupEnv("TEST_AWS_SES_VERIFIED_EMAIL_ARN")
	if !ok {
		t.Skip("'TEST_AWS_SES_VERIFIED_EMAIL_ARN' not set, skipping test.")
	}
	emailTo := sourceARN[strings.LastIndex(sourceARN, "/")+1:]

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t); testAccPreCheckAWSCognitoIdentityProvider(t) },
		ErrorCheck:   testAccErrorCheck(t, cognitoidentityprovider.EndpointsID),
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSCognitoUserPoolDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSCognitoUserPoolConfig_withEmailConfigurationSource(rName, replyTo, sourceARN, emailTo, "DEVELOPER"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "email_configuration.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "email_configuration.0.reply_to_email_address", replyTo),
					resource.TestCheckResourceAttr(resourceName, "email_configuration.0.email_sending_account", "DEVELOPER"),
					resource.TestCheckResourceAttr(resourceName, "email_configuration.0.source_arn", sourceARN),
					resource.TestCheckResourceAttr(resourceName, "email_configuration.0.from_email_address", emailTo),
					resource.TestCheckResourceAttrPair(resourceName, "email_configuration.0.configuration_set", resourceName2, "name"),
				),
			},
		},
	})
}

func TestAccAWSCognitoUserPool_withTags(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc-test")
	resourceName := "aws_cognito_user_pool.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t); testAccPreCheckAWSCognitoIdentityProvider(t) },
		ErrorCheck:   testAccErrorCheck(t, cognitoidentityprovider.EndpointsID),
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSCognitoUserPoolDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSCognitoUserPoolConfig_Tags1(rName, "key1", "value1"),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckAWSCognitoUserPoolExists(resourceName, nil),
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
				Config: testAccAWSCognitoUserPoolConfig_Tags2(rName, "key1", "value1updated", "key2", "value2"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "tags.%", "2"),
					resource.TestCheckResourceAttr(resourceName, "tags.key1", "value1updated"),
					resource.TestCheckResourceAttr(resourceName, "tags.key2", "value2"),
				),
			},
			{
				Config: testAccAWSCognitoUserPoolConfig_Tags1(rName, "key2", "value2"),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckAWSCognitoUserPoolExists(resourceName, nil),
					resource.TestCheckResourceAttr(resourceName, "tags.%", "1"),
					resource.TestCheckResourceAttr(resourceName, "tags.key2", "value2"),
				),
			},
		},
	})
}

func TestAccAWSCognitoUserPool_withAliasAttributes(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc-test")
	resourceName := "aws_cognito_user_pool.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t); testAccPreCheckAWSCognitoIdentityProvider(t) },
		ErrorCheck:   testAccErrorCheck(t, cognitoidentityprovider.EndpointsID),
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSCognitoUserPoolDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSCognitoUserPoolConfig_withAliasAttributes(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckAWSCognitoUserPoolExists(resourceName, nil),
					resource.TestCheckResourceAttr(resourceName, "alias_attributes.#", "1"),
					resource.TestCheckTypeSetElemAttr(resourceName, "alias_attributes.*", "preferred_username"),
					resource.TestCheckResourceAttr(resourceName, "auto_verified_attributes.#", "0"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: testAccAWSCognitoUserPoolConfig_withAliasAttributesUpdated(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "alias_attributes.#", "2"),
					resource.TestCheckTypeSetElemAttr(resourceName, "alias_attributes.*", "email"),
					resource.TestCheckTypeSetElemAttr(resourceName, "alias_attributes.*", "preferred_username"),
					resource.TestCheckResourceAttr(resourceName, "auto_verified_attributes.#", "1"),
					resource.TestCheckTypeSetElemAttr(resourceName, "auto_verified_attributes.*", "email"),
				),
			},
		},
	})
}

func TestAccAWSCognitoUserPool_withUsernameAttributes(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc-test")
	resourceName := "aws_cognito_user_pool.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t); testAccPreCheckAWSCognitoIdentityProvider(t) },
		ErrorCheck:   testAccErrorCheck(t, cognitoidentityprovider.EndpointsID),
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSCognitoUserPoolDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSCognitoUserPoolConfig_withUsernameAttributes(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckAWSCognitoUserPoolExists(resourceName, nil),
					resource.TestCheckResourceAttr(resourceName, "username_attributes.#", "1"),
					resource.TestCheckTypeSetElemAttr(resourceName, "username_attributes.*", "phone_number"),
					resource.TestCheckResourceAttr(resourceName, "auto_verified_attributes.#", "0"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: testAccAWSCognitoUserPoolConfig_withUsernameAttributesUpdated(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "username_attributes.#", "2"),
					resource.TestCheckTypeSetElemAttr(resourceName, "username_attributes.*", "email"),
					resource.TestCheckTypeSetElemAttr(resourceName, "username_attributes.*", "phone_number"),
					resource.TestCheckResourceAttr(resourceName, "auto_verified_attributes.#", "1"),
					resource.TestCheckTypeSetElemAttr(resourceName, "auto_verified_attributes.*", "email"),
				),
			},
		},
	})
}

func TestAccAWSCognitoUserPool_withPasswordPolicy(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc-test")
	resourceName := "aws_cognito_user_pool.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t); testAccPreCheckAWSCognitoIdentityProvider(t) },
		ErrorCheck:   testAccErrorCheck(t, cognitoidentityprovider.EndpointsID),
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSCognitoUserPoolDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSCognitoUserPoolConfig_withPasswordPolicy(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckAWSCognitoUserPoolExists(resourceName, nil),
					resource.TestCheckResourceAttr(resourceName, "password_policy.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "password_policy.0.minimum_length", "7"),
					resource.TestCheckResourceAttr(resourceName, "password_policy.0.require_lowercase", "true"),
					resource.TestCheckResourceAttr(resourceName, "password_policy.0.require_numbers", "false"),
					resource.TestCheckResourceAttr(resourceName, "password_policy.0.require_symbols", "true"),
					resource.TestCheckResourceAttr(resourceName, "password_policy.0.require_uppercase", "false"),
					resource.TestCheckResourceAttr(resourceName, "password_policy.0.temporary_password_validity_days", "7"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: testAccAWSCognitoUserPoolConfig_withPasswordPolicyUpdated(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "password_policy.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "password_policy.0.minimum_length", "9"),
					resource.TestCheckResourceAttr(resourceName, "password_policy.0.require_lowercase", "false"),
					resource.TestCheckResourceAttr(resourceName, "password_policy.0.require_numbers", "true"),
					resource.TestCheckResourceAttr(resourceName, "password_policy.0.require_symbols", "false"),
					resource.TestCheckResourceAttr(resourceName, "password_policy.0.require_uppercase", "true"),
					resource.TestCheckResourceAttr(resourceName, "password_policy.0.temporary_password_validity_days", "14"),
				),
			},
		},
	})
}

func TestAccAWSCognitoUserPool_withUsernameConfiguration(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc-test")
	resourceName := "aws_cognito_user_pool.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t); testAccPreCheckAWSCognitoIdentityProvider(t) },
		ErrorCheck:   testAccErrorCheck(t, cognitoidentityprovider.EndpointsID),
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSCognitoUserPoolDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSCognitoUserPoolConfig_withUsernameConfiguration(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckAWSCognitoUserPoolExists(resourceName, nil),
					resource.TestCheckResourceAttr(resourceName, "username_configuration.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "username_configuration.0.case_sensitive", "true"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: testAccAWSCognitoUserPoolConfig_withUsernameConfigurationUpdated(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckAWSCognitoUserPoolExists(resourceName, nil),
					resource.TestCheckResourceAttr(resourceName, "username_configuration.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "username_configuration.0.case_sensitive", "false"),
				),
			},
		},
	})
}

func TestAccAWSCognitoUserPool_withLambdaConfig(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc-test")
	resourceName := "aws_cognito_user_pool.test"
	lambdaResourceName := "aws_lambda_function.test"
	lambdaUpdatedResourceName := "aws_lambda_function.second"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t); testAccPreCheckAWSCognitoIdentityProvider(t) },
		ErrorCheck:   testAccErrorCheck(t, cognitoidentityprovider.EndpointsID),
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSCognitoUserPoolDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSCognitoUserPoolConfig_withLambdaConfig(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckAWSCognitoUserPoolExists(resourceName, nil),
					resource.TestCheckResourceAttr(resourceName, "lambda_config.#", "1"),
					resource.TestCheckResourceAttrPair(resourceName, "lambda_config.0.create_auth_challenge", lambdaResourceName, "arn"),
					resource.TestCheckResourceAttrPair(resourceName, "lambda_config.0.custom_message", lambdaResourceName, "arn"),
					resource.TestCheckResourceAttrPair(resourceName, "lambda_config.0.define_auth_challenge", lambdaResourceName, "arn"),
					resource.TestCheckResourceAttrPair(resourceName, "lambda_config.0.post_authentication", lambdaResourceName, "arn"),
					resource.TestCheckResourceAttrPair(resourceName, "lambda_config.0.post_confirmation", lambdaResourceName, "arn"),
					resource.TestCheckResourceAttrPair(resourceName, "lambda_config.0.pre_authentication", lambdaResourceName, "arn"),
					resource.TestCheckResourceAttrPair(resourceName, "lambda_config.0.pre_sign_up", lambdaResourceName, "arn"),
					resource.TestCheckResourceAttrPair(resourceName, "lambda_config.0.pre_token_generation", lambdaResourceName, "arn"),
					resource.TestCheckResourceAttrPair(resourceName, "lambda_config.0.user_migration", lambdaResourceName, "arn"),
					resource.TestCheckResourceAttrPair(resourceName, "lambda_config.0.verify_auth_challenge_response", lambdaResourceName, "arn"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: testAccAWSCognitoUserPoolConfig_withLambdaConfigUpdated(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "lambda_config.#", "1"),
					resource.TestCheckResourceAttrPair(resourceName, "lambda_config.0.create_auth_challenge", lambdaUpdatedResourceName, "arn"),
					resource.TestCheckResourceAttrPair(resourceName, "lambda_config.0.custom_message", lambdaUpdatedResourceName, "arn"),
					resource.TestCheckResourceAttrPair(resourceName, "lambda_config.0.define_auth_challenge", lambdaUpdatedResourceName, "arn"),
					resource.TestCheckResourceAttrPair(resourceName, "lambda_config.0.post_authentication", lambdaUpdatedResourceName, "arn"),
					resource.TestCheckResourceAttrPair(resourceName, "lambda_config.0.post_confirmation", lambdaUpdatedResourceName, "arn"),
					resource.TestCheckResourceAttrPair(resourceName, "lambda_config.0.pre_authentication", lambdaUpdatedResourceName, "arn"),
					resource.TestCheckResourceAttrPair(resourceName, "lambda_config.0.pre_sign_up", lambdaUpdatedResourceName, "arn"),
					resource.TestCheckResourceAttrPair(resourceName, "lambda_config.0.pre_token_generation", lambdaUpdatedResourceName, "arn"),
					resource.TestCheckResourceAttrPair(resourceName, "lambda_config.0.user_migration", lambdaUpdatedResourceName, "arn"),
					resource.TestCheckResourceAttrPair(resourceName, "lambda_config.0.verify_auth_challenge_response", lambdaUpdatedResourceName, "arn"),
				),
			},
			{
				Config: testAccAWSCognitoUserPoolConfig_Name(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckAWSCognitoUserPoolExists(resourceName, nil),
					resource.TestCheckResourceAttr(resourceName, "lambda_config.#", "0"),
				),
			},
		},
	})
}

func testAccCheckAWSCognitoUserPoolNotRecreated(pool1, pool2 *cognitoidentityprovider.DescribeUserPoolOutput) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if !aws.TimeValue(pool1.UserPool.CreationDate).Equal(aws.TimeValue(pool2.UserPool.CreationDate)) {
			return fmt.Errorf("user pool was recreated. expected: %s, got: %s", pool1.UserPool.CreationDate, pool2.UserPool.CreationDate)
		}
		return nil
	}
}

func TestAccAWSCognitoUserPool_withLambdaConfig_emailConfig(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc-test")
	resourceName := "aws_cognito_user_pool.test"
	lambdaResourceName := "aws_lambda_function.test"
	lambdaUpdatedResourceName := "aws_lambda_function.second"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t); testAccPreCheckAWSCognitoIdentityProvider(t) },
		ErrorCheck:   testAccErrorCheck(t, cognitoidentityprovider.EndpointsID),
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSCognitoUserPoolDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSCognitoUserPoolLambdaConfigEmailSenderConfig(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckAWSCognitoUserPoolExists(resourceName, nil),
					resource.TestCheckResourceAttr(resourceName, "lambda_config.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "lambda_config.0.custom_email_sender.#", "1"),
					resource.TestCheckResourceAttrPair(resourceName, "lambda_config.0.custom_email_sender.0.lambda_arn", lambdaResourceName, "arn"),
					resource.TestCheckResourceAttr(resourceName, "lambda_config.0.custom_email_sender.0.lambda_version", "V1_0"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: testAccAWSCognitoUserPoolLambdaConfigEmailSenderConfigUpdated(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "lambda_config.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "lambda_config.0.custom_email_sender.#", "1"),
					resource.TestCheckResourceAttrPair(resourceName, "lambda_config.0.custom_email_sender.0.lambda_arn", lambdaUpdatedResourceName, "arn"),
					resource.TestCheckResourceAttr(resourceName, "lambda_config.0.custom_email_sender.0.lambda_version", "V1_0"),
				),
			},
		},
	})
}

func TestAccAWSCognitoUserPool_withLambdaConfig_smsConfig(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc-test")
	resourceName := "aws_cognito_user_pool.test"
	lambdaResourceName := "aws_lambda_function.test"
	lambdaUpdatedResourceName := "aws_lambda_function.second"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t); testAccPreCheckAWSCognitoIdentityProvider(t) },
		ErrorCheck:   testAccErrorCheck(t, cognitoidentityprovider.EndpointsID),
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSCognitoUserPoolDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSCognitoUserPoolLambdaConfigSMSSenderConfig(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckAWSCognitoUserPoolExists(resourceName, nil),
					resource.TestCheckResourceAttr(resourceName, "lambda_config.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "lambda_config.0.custom_sms_sender.#", "1"),
					resource.TestCheckResourceAttrPair(resourceName, "lambda_config.0.custom_sms_sender.0.lambda_arn", lambdaResourceName, "arn"),
					resource.TestCheckResourceAttr(resourceName, "lambda_config.0.custom_sms_sender.0.lambda_version", "V1_0"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: testAccAWSCognitoUserPoolLambdaConfigSMSSenderConfigUpdated(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "lambda_config.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "lambda_config.0.custom_sms_sender.#", "1"),
					resource.TestCheckResourceAttrPair(resourceName, "lambda_config.0.custom_sms_sender.0.lambda_arn", lambdaUpdatedResourceName, "arn"),
					resource.TestCheckResourceAttr(resourceName, "lambda_config.0.custom_sms_sender.0.lambda_version", "V1_0"),
				),
			},
		},
	})
}

func TestAccAWSCognitoUserPool_schemaAttributes(t *testing.T) {
	var pool1, pool2 cognitoidentityprovider.DescribeUserPoolOutput
	rName := acctest.RandomWithPrefix("tf-acc-test")
	resourceName := "aws_cognito_user_pool.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t); testAccPreCheckAWSCognitoIdentityProvider(t) },
		ErrorCheck:   testAccErrorCheck(t, cognitoidentityprovider.EndpointsID),
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSCognitoUserPoolDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSCognitoUserPoolConfig_withSchemaAttributes(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckAWSCognitoUserPoolExists(resourceName, &pool1),
					resource.TestCheckResourceAttr(resourceName, "schema.#", "2"),
					resource.TestCheckTypeSetElemNestedAttrs(resourceName, "schema.*", map[string]string{
						"attribute_data_type":            "String",
						"developer_only_attribute":       "false",
						"mutable":                        "false",
						"name":                           "email",
						"number_attribute_constraints.#": "0",
						"required":                       "true",
						"string_attribute_constraints.#": "1",
						"string_attribute_constraints.0.min_length": "5",
						"string_attribute_constraints.0.max_length": "10",
					}),
					resource.TestCheckTypeSetElemNestedAttrs(resourceName, "schema.*", map[string]string{
						"attribute_data_type":            "Boolean",
						"developer_only_attribute":       "true",
						"mutable":                        "false",
						"name":                           "mybool",
						"number_attribute_constraints.#": "0",
						"required":                       "false",
						"string_attribute_constraints.#": "0",
					}),
				),
			},
			{
				Config: testAccAWSCognitoUserPoolConfig_withSchemaAttributesUpdated(rName, "mybool"),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckAWSCognitoUserPoolExists(resourceName, &pool2),
					testAccCheckAWSCognitoUserPoolNotRecreated(&pool1, &pool2),
					resource.TestCheckResourceAttr(resourceName, "schema.#", "3"),
					resource.TestCheckTypeSetElemNestedAttrs(resourceName, "schema.*", map[string]string{
						"attribute_data_type":            "String",
						"developer_only_attribute":       "false",
						"mutable":                        "false",
						"name":                           "email",
						"number_attribute_constraints.#": "0",
						"required":                       "true",
						"string_attribute_constraints.#": "1",
						"string_attribute_constraints.0.min_length": "5",
						"string_attribute_constraints.0.max_length": "10",
					}),
					resource.TestCheckTypeSetElemNestedAttrs(resourceName, "schema.*", map[string]string{
						"attribute_data_type":            "Boolean",
						"developer_only_attribute":       "true",
						"mutable":                        "false",
						"name":                           "mybool",
						"number_attribute_constraints.#": "0",
						"required":                       "false",
						"string_attribute_constraints.#": "0",
					}),
					resource.TestCheckTypeSetElemNestedAttrs(resourceName, "schema.*", map[string]string{
						"attribute_data_type":            "Number",
						"developer_only_attribute":       "false",
						"mutable":                        "true",
						"name":                           "mynondevnumber",
						"number_attribute_constraints.#": "1",
						"number_attribute_constraints.0.min_value": "2",
						"number_attribute_constraints.0.max_value": "6",
						"required":                       "false",
						"string_attribute_constraints.#": "0",
					}),
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

func TestAccAWSCognitoUserPool_schemaAttributesRemoved(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc-test")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t); testAccPreCheckAWSCognitoIdentityProvider(t) },
		ErrorCheck:   testAccErrorCheck(t, cognitoidentityprovider.EndpointsID),
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSCognitoUserPoolDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSCognitoUserPoolConfig_withSchemaAttributesUpdated(rName, "mybool"),
			},
			{
				Config:      testAccAWSCognitoUserPoolConfig_withSchemaAttributes(rName),
				ExpectError: regexp.MustCompile("cannot modify or remove schema items"),
			},
		},
	})
}

func TestAccAWSCognitoUserPool_schemaAttributesModified(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc-test")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t); testAccPreCheckAWSCognitoIdentityProvider(t) },
		ErrorCheck:   testAccErrorCheck(t, cognitoidentityprovider.EndpointsID),
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSCognitoUserPoolDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSCognitoUserPoolConfig_withSchemaAttributesUpdated(rName, "mybool"),
			},
			{
				Config:      testAccAWSCognitoUserPoolConfig_withSchemaAttributesUpdated(rName, "mybool2"),
				ExpectError: regexp.MustCompile("cannot modify or remove schema items"),
			},
		},
	})
}

func TestAccAWSCognitoUserPool_withVerificationMessageTemplate(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc-test")
	resourceName := "aws_cognito_user_pool.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t); testAccPreCheckAWSCognitoIdentityProvider(t) },
		ErrorCheck:   testAccErrorCheck(t, cognitoidentityprovider.EndpointsID),
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSCognitoUserPoolDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSCognitoUserPoolConfig_withVerificationMessageTemplate(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckAWSCognitoUserPoolExists(resourceName, nil),
					resource.TestCheckResourceAttr(resourceName, "verification_message_template.0.default_email_option", "CONFIRM_WITH_LINK"),
					resource.TestCheckResourceAttr(resourceName, "verification_message_template.0.email_message", "foo {####} bar"),
					resource.TestCheckResourceAttr(resourceName, "verification_message_template.0.email_message_by_link", "{##foobar##}"),
					resource.TestCheckResourceAttr(resourceName, "verification_message_template.0.email_subject", "foobar {####}"),
					resource.TestCheckResourceAttr(resourceName, "verification_message_template.0.email_subject_by_link", "foobar"),
					resource.TestCheckResourceAttr(resourceName, "verification_message_template.0.sms_message", "{####} baz"),

					/* Setting Verification template attributes like EmailMessage, EmailSubject or SmsMessage
					will implicitly set EmailVerificationMessage, EmailVerificationSubject and SmsVerificationMessage attributes.
					*/
					resource.TestCheckResourceAttr(resourceName, "email_verification_message", "foo {####} bar"),
					resource.TestCheckResourceAttr(resourceName, "email_verification_subject", "foobar {####}"),
					resource.TestCheckResourceAttr(resourceName, "sms_verification_message", "{####} baz"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: testAccAWSCognitoUserPoolConfig_withVerificationMessageTemplate_DefaultEmailOption(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "verification_message_template.0.default_email_option", "CONFIRM_WITH_CODE"),
					resource.TestCheckResourceAttr(resourceName, "email_verification_message", "{####} Baz"),
					resource.TestCheckResourceAttr(resourceName, "email_verification_subject", "BazBaz {####}"),
					resource.TestCheckResourceAttr(resourceName, "sms_verification_message", "{####} BazBazBar?"),

					/* Setting EmailVerificationMessage, EmailVerificationSubject and SmsVerificationMessage attributes
					will implicitly set verification template attributes like EmailMessage, EmailSubject or SmsMessage.
					*/
					resource.TestCheckResourceAttr(resourceName, "verification_message_template.0.email_message", "{####} Baz"),
					resource.TestCheckResourceAttr(resourceName, "verification_message_template.0.email_subject", "BazBaz {####}"),
					resource.TestCheckResourceAttr(resourceName, "verification_message_template.0.sms_message", "{####} BazBazBar?"),
				),
			},
		},
	})
}

func TestAccAWSCognitoUserPool_update(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc-test")
	optionalMfa := "OPTIONAL"
	offMfa := "OFF"
	authenticationMessage := fmt.Sprintf("%s {####}", acctest.RandString(10))
	updatedAuthenticationMessage := fmt.Sprintf("%s {####}", acctest.RandString(10))
	resourceName := "aws_cognito_user_pool.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t); testAccPreCheckAWSCognitoIdentityProvider(t) },
		ErrorCheck:   testAccErrorCheck(t, cognitoidentityprovider.EndpointsID),
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSCognitoUserPoolDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSCognitoUserPoolConfig_update(rName, optionalMfa, authenticationMessage),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckAWSCognitoUserPoolExists(resourceName, nil),
					resource.TestCheckResourceAttr(resourceName, "auto_verified_attributes.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "mfa_configuration", optionalMfa),
					resource.TestCheckResourceAttr(resourceName, "email_verification_message", "Foo {####} Bar"),
					resource.TestCheckResourceAttr(resourceName, "email_verification_subject", "FooBar {####}"),
					resource.TestCheckResourceAttr(resourceName, "sms_verification_message", "{####} Baz"),
					resource.TestCheckResourceAttr(resourceName, "sms_authentication_message", authenticationMessage),
					resource.TestCheckResourceAttr(resourceName, "admin_create_user_config.0.allow_admin_create_user_only", "true"),
					resource.TestCheckResourceAttr(resourceName, "admin_create_user_config.0.invite_message_template.0.email_message", "Your username is {username} and temporary password is {####}. "),
					resource.TestCheckResourceAttr(resourceName, "admin_create_user_config.0.invite_message_template.0.email_subject", "FooBar {####}"),
					resource.TestCheckResourceAttr(resourceName, "admin_create_user_config.0.invite_message_template.0.sms_message", "Your username is {username} and temporary password is {####}."),
					resource.TestCheckResourceAttr(resourceName, "device_configuration.0.challenge_required_on_new_device", "true"),
					resource.TestCheckResourceAttr(resourceName, "device_configuration.0.device_only_remembered_on_user_prompt", "false"),
					resource.TestCheckResourceAttr(resourceName, "verification_message_template.0.default_email_option", "CONFIRM_WITH_CODE"),
					resource.TestCheckResourceAttr(resourceName, "sms_configuration.#", "1"),
					resource.TestCheckResourceAttrSet(resourceName, "sms_configuration.0.external_id"),
					resource.TestCheckResourceAttrSet(resourceName, "sms_configuration.0.sns_caller_arn"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: testAccAWSCognitoUserPoolConfig_update(rName, optionalMfa, updatedAuthenticationMessage),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckAWSCognitoUserPoolExists(resourceName, nil),
					resource.TestCheckResourceAttr(resourceName, "auto_verified_attributes.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "mfa_configuration", optionalMfa),
					resource.TestCheckResourceAttr(resourceName, "email_verification_message", "Foo {####} Bar"),
					resource.TestCheckResourceAttr(resourceName, "email_verification_subject", "FooBar {####}"),
					resource.TestCheckResourceAttr(resourceName, "sms_verification_message", "{####} Baz"),
					resource.TestCheckResourceAttr(resourceName, "sms_authentication_message", updatedAuthenticationMessage),
					resource.TestCheckResourceAttr(resourceName, "admin_create_user_config.0.allow_admin_create_user_only", "true"),
					resource.TestCheckResourceAttr(resourceName, "admin_create_user_config.0.invite_message_template.0.email_message", "Your username is {username} and temporary password is {####}. "),
					resource.TestCheckResourceAttr(resourceName, "admin_create_user_config.0.invite_message_template.0.email_subject", "FooBar {####}"),
					resource.TestCheckResourceAttr(resourceName, "admin_create_user_config.0.invite_message_template.0.sms_message", "Your username is {username} and temporary password is {####}."),
					resource.TestCheckResourceAttr(resourceName, "device_configuration.0.challenge_required_on_new_device", "true"),
					resource.TestCheckResourceAttr(resourceName, "device_configuration.0.device_only_remembered_on_user_prompt", "false"),
					resource.TestCheckResourceAttr(resourceName, "verification_message_template.0.default_email_option", "CONFIRM_WITH_CODE"),
					resource.TestCheckResourceAttr(resourceName, "sms_configuration.#", "1"),
					resource.TestCheckResourceAttrSet(resourceName, "sms_configuration.0.external_id"),
					resource.TestCheckResourceAttrSet(resourceName, "sms_configuration.0.sns_caller_arn"),
				),
			},
			{
				Config: testAccAWSCognitoUserPoolConfig_update(rName, offMfa, updatedAuthenticationMessage),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckAWSCognitoUserPoolExists(resourceName, nil),
					resource.TestCheckResourceAttr(resourceName, "auto_verified_attributes.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "mfa_configuration", offMfa),
					resource.TestCheckResourceAttr(resourceName, "email_verification_message", "Foo {####} Bar"),
					resource.TestCheckResourceAttr(resourceName, "email_verification_subject", "FooBar {####}"),
					resource.TestCheckResourceAttr(resourceName, "sms_verification_message", "{####} Baz"),
					resource.TestCheckResourceAttr(resourceName, "sms_authentication_message", updatedAuthenticationMessage),
					resource.TestCheckResourceAttr(resourceName, "admin_create_user_config.0.allow_admin_create_user_only", "true"),
					resource.TestCheckResourceAttr(resourceName, "admin_create_user_config.0.invite_message_template.0.email_message", "Your username is {username} and temporary password is {####}. "),
					resource.TestCheckResourceAttr(resourceName, "admin_create_user_config.0.invite_message_template.0.email_subject", "FooBar {####}"),
					resource.TestCheckResourceAttr(resourceName, "admin_create_user_config.0.invite_message_template.0.sms_message", "Your username is {username} and temporary password is {####}."),
					resource.TestCheckResourceAttr(resourceName, "device_configuration.0.challenge_required_on_new_device", "true"),
					resource.TestCheckResourceAttr(resourceName, "device_configuration.0.device_only_remembered_on_user_prompt", "false"),
					resource.TestCheckResourceAttr(resourceName, "verification_message_template.0.default_email_option", "CONFIRM_WITH_CODE"),
					resource.TestCheckResourceAttr(resourceName, "sms_configuration.#", "1"),
					resource.TestCheckResourceAttrSet(resourceName, "sms_configuration.0.external_id"),
					resource.TestCheckResourceAttrSet(resourceName, "sms_configuration.0.sns_caller_arn"),
				),
			},
		},
	})
}

func TestAccAWSCognitoUserPool_disappears(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc-test")
	resourceName := "aws_cognito_user_pool.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t); testAccPreCheckAWSCognitoIdentityProvider(t) },
		ErrorCheck:   testAccErrorCheck(t, cognitoidentityprovider.EndpointsID),
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSCognitoUserPoolDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSCognitoUserPoolConfig_Name(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckAWSCognitoUserPoolExists(resourceName, nil),
					testAccCheckResourceDisappears(testAccProvider, resourceAwsCognitoUserPool(), resourceName),
				),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func testAccCheckAWSCognitoUserPoolDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*AWSClient).cognitoidpconn

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "aws_cognito_user_pool" {
			continue
		}

		params := &cognitoidentityprovider.DescribeUserPoolInput{
			UserPoolId: aws.String(rs.Primary.ID),
		}

		_, err := conn.DescribeUserPool(params)

		if err != nil {
			if isAWSErr(err, cognitoidentityprovider.ErrCodeResourceNotFoundException, "") {
				return nil
			}
			return err
		}
	}

	return nil
}

func testAccCheckAWSCognitoUserPoolExists(name string, pool *cognitoidentityprovider.DescribeUserPoolOutput) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("Not found: %s", name)
		}

		if rs.Primary.ID == "" {
			return errors.New("No Cognito User Pool ID set")
		}

		conn := testAccProvider.Meta().(*AWSClient).cognitoidpconn

		params := &cognitoidentityprovider.DescribeUserPoolInput{
			UserPoolId: aws.String(rs.Primary.ID),
		}

		poolOut, err := conn.DescribeUserPool(params)
		if err != nil {
			return err
		}

		if pool != nil {
			*pool = *poolOut
		}

		return nil
	}
}

func testAccPreCheckAWSCognitoIdentityProvider(t *testing.T) {
	conn := testAccProvider.Meta().(*AWSClient).cognitoidpconn

	input := &cognitoidentityprovider.ListUserPoolsInput{
		MaxResults: aws.Int64(int64(1)),
	}

	_, err := conn.ListUserPools(input)

	if testAccPreCheckSkipError(err) {
		t.Skipf("skipping acceptance testing: %s", err)
	}

	if err != nil {
		t.Fatalf("unexpected PreCheck error: %s", err)
	}
}

func testAccAWSCognitoUserPoolConfigSmsConfigurationBase(rName string, externalID string) string {
	return fmt.Sprintf(`
data "aws_partition" "current" {}

resource "aws_iam_role" "test" {
  name = %[1]q

  assume_role_policy = jsonencode({
    Statement = [{
      Action = "sts:AssumeRole"
      Condition = {
        "StringEquals" = {
          "sts:ExternalId" = %[2]q
        }
      }

      Effect = "Allow"
      Principal = {
        Service = "cognito-idp.${data.aws_partition.current.dns_suffix}"
      }
    }]
    Version = "2012-10-17"
  })
}

resource "aws_iam_role_policy" "test" {
  role = aws_iam_role.test.id

  policy = jsonencode({
    Statement = [{
      Action   = "sns:publish"
      Effect   = "Allow"
      Resource = "*"
    }]
    Version = "2012-10-17"
  })
}
`, rName, externalID)
}

func testAccAWSCognitoUserPoolConfig_Name(rName string) string {
	return fmt.Sprintf(`
resource "aws_cognito_user_pool" "test" {
  name = %[1]q
}
`, rName)
}

func testAccAWSCognitoUserPoolConfigAccountRecoverySingle(rName string) string {
	return fmt.Sprintf(`
resource "aws_cognito_user_pool" "test" {
  name = %[1]q

  account_recovery_setting {
    recovery_mechanism {
      name     = "verified_email"
      priority = 1
    }
  }
}
`, rName)
}

func testAccAWSCognitoUserPoolConfigAccountRecoveryMulti(rName string) string {
	return fmt.Sprintf(`
resource "aws_cognito_user_pool" "test" {
  name = %[1]q

  account_recovery_setting {
    recovery_mechanism {
      name     = "verified_email"
      priority = 1
    }

    recovery_mechanism {
      name     = "verified_phone_number"
      priority = 2
    }
  }
}
`, rName)
}

func testAccAWSCognitoUserPoolConfigAccountRecoveryUpdate(rName string) string {
	return fmt.Sprintf(`
resource "aws_cognito_user_pool" "test" {
  name = %[1]q

  account_recovery_setting {
    recovery_mechanism {
      name     = "verified_phone_number"
      priority = 1
    }
  }
}
`, rName)
}

func testAccAWSCognitoUserPoolConfig_withAdminCreateUserConfiguration(rName string) string {
	return fmt.Sprintf(`
resource "aws_cognito_user_pool" "test" {
  name = %[1]q

  admin_create_user_config {
    allow_admin_create_user_only = true

    invite_message_template {
      email_message = "Your username is {username} and temporary password is {####}. "
      email_subject = "FooBar {####}"
      sms_message   = "Your username is {username} and temporary password is {####}."
    }
  }
}
`, rName)
}

func testAccAWSCognitoUserPoolConfig_withAdminCreateUserConfigurationUpdated(rName string) string {
	return fmt.Sprintf(`
resource "aws_cognito_user_pool" "test" {
  name = %[1]q

  admin_create_user_config {
    allow_admin_create_user_only = false

    invite_message_template {
      email_message = "Your username is {username} and constant password is {####}. "
      email_subject = "Foo{####}BaBaz"
      sms_message   = "Your username is {username} and constant password is {####}."
    }
  }
}
`, rName)
}

func testAccAWSCognitoUserPoolConfig_AdvancedSecurityMode(rName string, advancedSecurityMode string) string {
	return fmt.Sprintf(`
resource "aws_cognito_user_pool" "test" {
  name = %[1]q

  user_pool_add_ons {
    advanced_security_mode = %[2]q
  }
}
`, rName, advancedSecurityMode)
}

func testAccAWSCognitoUserPoolConfig_withDeviceConfiguration(rName string) string {
	return fmt.Sprintf(`
resource "aws_cognito_user_pool" "test" {
  name = %[1]q

  device_configuration {
    challenge_required_on_new_device      = true
    device_only_remembered_on_user_prompt = false
  }
}
`, rName)
}

func testAccAWSCognitoUserPoolConfig_withDeviceConfigurationUpdated(rName string) string {
	return fmt.Sprintf(`
resource "aws_cognito_user_pool" "test" {
  name = %[1]q

  device_configuration {
    challenge_required_on_new_device      = false
    device_only_remembered_on_user_prompt = true
  }
}
`, rName)
}

func testAccAWSCognitoUserPoolConfig_withEmailVerificationMessage(rName, subject, message string) string {
	return fmt.Sprintf(`
resource "aws_cognito_user_pool" "test" {
  name                       = %[1]q
  email_verification_subject = "%[2]s"
  email_verification_message = "%[3]s"

  verification_message_template {
    default_email_option = "CONFIRM_WITH_CODE"
  }
}
`, rName, subject, message)
}

func testAccAWSCognitoUserPoolConfig_MfaConfiguration(rName string, mfaConfiguration string) string {
	return fmt.Sprintf(`
resource "aws_cognito_user_pool" "test" {
  mfa_configuration = %[2]q
  name              = %[1]q
}
`, rName, mfaConfiguration)
}

func testAccAWSCognitoUserPoolConfig_MfaConfiguration_SmsConfiguration(rName string) string {
	return testAccAWSCognitoUserPoolConfigSmsConfigurationBase(rName, "test") + fmt.Sprintf(`
resource "aws_cognito_user_pool" "test" {
  mfa_configuration = "ON"
  name              = %[1]q

  sms_configuration {
    external_id    = "test"
    sns_caller_arn = aws_iam_role.test.arn
  }
}
`, rName)
}

func testAccAWSCognitoUserPoolConfig_MfaConfiguration_SmsConfigurationAndSoftwareTokenMfaConfigurationEnabled(rName string, enabled bool) string {
	return testAccAWSCognitoUserPoolConfigSmsConfigurationBase(rName, "test") + fmt.Sprintf(`
resource "aws_cognito_user_pool" "test" {
  mfa_configuration = "ON"
  name              = %[1]q

  sms_configuration {
    external_id    = "test"
    sns_caller_arn = aws_iam_role.test.arn
  }

  software_token_mfa_configuration {
    enabled = %[2]t
  }
}
`, rName, enabled)
}

func testAccAWSCognitoUserPoolConfig_MfaConfiguration_SoftwareTokenMfaConfigurationEnabled(rName string, enabled bool) string {
	return fmt.Sprintf(`
resource "aws_cognito_user_pool" "test" {
  mfa_configuration = "ON"
  name              = %[1]q

  software_token_mfa_configuration {
    enabled = %[2]t
  }
}
`, rName, enabled)
}

func testAccAWSCognitoUserPoolConfig_SmsAuthenticationMessage(rName, smsAuthenticationMessage string) string {
	return fmt.Sprintf(`
resource "aws_cognito_user_pool" "test" {
  name                       = %[1]q
  sms_authentication_message = %[2]q
}
`, rName, smsAuthenticationMessage)
}

func testAccAWSCognitoUserPoolConfig_SmsConfiguration_ExternalId(rName string, externalID string) string {
	return testAccAWSCognitoUserPoolConfigSmsConfigurationBase(rName, externalID) + fmt.Sprintf(`
resource "aws_cognito_user_pool" "test" {
  name = %[1]q

  sms_configuration {
    external_id    = %[2]q
    sns_caller_arn = aws_iam_role.test.arn
  }
}
`, rName, externalID)
}

func testAccAWSCognitoUserPoolConfig_SmsConfiguration_SnsCallerArn2(rName string) string {
	return testAccAWSCognitoUserPoolConfigSmsConfigurationBase(rName+"-2", "test") + fmt.Sprintf(`
resource "aws_cognito_user_pool" "test" {
  name = %[1]q

  sms_configuration {
    external_id    = "test"
    sns_caller_arn = aws_iam_role.test.arn
  }
}
`, rName)
}

func testAccAWSCognitoUserPoolConfig_SmsVerificationMessage(rName, smsVerificationMessage string) string {
	return fmt.Sprintf(`
resource "aws_cognito_user_pool" "test" {
  name                     = %[1]q
  sms_verification_message = %[2]q
}
`, rName, smsVerificationMessage)
}

func testAccAWSCognitoUserPoolConfig_Tags1(rName, tagKey1, tagValue1 string) string {
	return fmt.Sprintf(`
resource "aws_cognito_user_pool" "test" {
  name = %[1]q

  tags = {
    %[2]q = %[3]q
  }
}
`, rName, tagKey1, tagValue1)
}

func testAccAWSCognitoUserPoolConfig_Tags2(rName, tagKey1, tagValue1, tagKey2, tagValue2 string) string {
	return fmt.Sprintf(`
resource "aws_cognito_user_pool" "test" {
  name = %[1]q

  tags = {
    %[2]q = %[3]q
    %[4]q = %[5]q
  }
}
`, rName, tagKey1, tagValue1, tagKey2, tagValue2)
}

func testAccAWSCognitoUserPoolConfig_withEmailConfiguration(rName string) string {
	return fmt.Sprintf(`
resource "aws_cognito_user_pool" "test" {
  name = %[1]q

  email_configuration {
    email_sending_account = "COGNITO_DEFAULT"
  }
}
`, rName)
}

func testAccAWSCognitoUserPoolConfig_withEmailConfigurationSource(rName, email, arn, from, account string) string {
	return fmt.Sprintf(`
resource "aws_ses_configuration_set" "test" {
  name = %[1]q

  delivery_options {
    tls_policy = "Optional"
  }
}

resource "aws_cognito_user_pool" "test" {
  name = %[1]q

  email_configuration {
    reply_to_email_address = %[2]q
    source_arn             = %[3]q
    from_email_address     = %[4]q
    email_sending_account  = %[5]q
    configuration_set      = aws_ses_configuration_set.test.name
  }
}
`, rName, email, arn, from, account)
}

func testAccAWSCognitoUserPoolConfig_withAliasAttributes(rName string) string {
	return fmt.Sprintf(`
resource "aws_cognito_user_pool" "test" {
  name = %[1]q

  alias_attributes = ["preferred_username"]
}
`, rName)
}

func testAccAWSCognitoUserPoolConfig_withAliasAttributesUpdated(rName string) string {
	return fmt.Sprintf(`
resource "aws_cognito_user_pool" "test" {
  name = %[1]q

  alias_attributes         = ["email", "preferred_username"]
  auto_verified_attributes = ["email"]
}
`, rName)
}

func testAccAWSCognitoUserPoolConfig_withUsernameAttributes(rName string) string {
	return fmt.Sprintf(`
resource "aws_cognito_user_pool" "test" {
  name = %[1]q

  username_attributes = ["phone_number"]
}
`, rName)
}

func testAccAWSCognitoUserPoolConfig_withUsernameAttributesUpdated(rName string) string {
	return fmt.Sprintf(`
resource "aws_cognito_user_pool" "test" {
  name = %[1]q

  username_attributes      = ["email", "phone_number"]
  auto_verified_attributes = ["email"]
}
`, rName)
}

func testAccAWSCognitoUserPoolConfig_withAdminCreateUserConfigAndPasswordPolicy(rName string) string {
	return fmt.Sprintf(`
resource "aws_cognito_user_pool" "test" {
  name = %[1]q

  admin_create_user_config {
    allow_admin_create_user_only = true
  }

  password_policy {
    minimum_length                   = 7
    require_lowercase                = true
    require_numbers                  = false
    require_symbols                  = true
    require_uppercase                = false
    temporary_password_validity_days = 7
  }
}
`, rName)
}

func testAccAWSCognitoUserPoolConfig_withPasswordPolicy(name string) string {
	return fmt.Sprintf(`
resource "aws_cognito_user_pool" "test" {
  name = %[1]q

  password_policy {
    minimum_length                   = 7
    require_lowercase                = true
    require_numbers                  = false
    require_symbols                  = true
    require_uppercase                = false
    temporary_password_validity_days = 7
  }
}
`, name)
}

func testAccAWSCognitoUserPoolConfig_withPasswordPolicyUpdated(name string) string {
	return fmt.Sprintf(`
resource "aws_cognito_user_pool" "test" {
  name = %[1]q

  password_policy {
    minimum_length                   = 9
    require_lowercase                = false
    require_numbers                  = true
    require_symbols                  = false
    require_uppercase                = true
    temporary_password_validity_days = 14
  }
}
`, name)
}

func testAccAWSCognitoUserPoolConfig_withUsernameConfiguration(name string) string {
	return fmt.Sprintf(`
resource "aws_cognito_user_pool" "test" {
  name = %[1]q

  username_configuration {
    case_sensitive = true
  }
}
`, name)
}

func testAccAWSCognitoUserPoolConfig_withUsernameConfigurationUpdated(name string) string {
	return fmt.Sprintf(`
resource "aws_cognito_user_pool" "test" {
  name = %[1]q

  username_configuration {
    case_sensitive = false
  }
}
`, name)
}

func testAccAWSCognitoUserPoolConfigLambdaConfigBase(name string) string {
	return fmt.Sprintf(`
resource "aws_iam_role" "test" {
  name = %[1]q

  assume_role_policy = <<EOF
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Action": "sts:AssumeRole",
      "Principal": {
        "Service": "lambda.amazonaws.com"
      },
      "Effect": "Allow",
      "Sid": ""
    }
  ]
}
EOF
}

resource "aws_lambda_function" "test" {
  filename      = "test-fixtures/lambdatest.zip"
  function_name = %[1]q
  role          = aws_iam_role.test.arn
  handler       = "exports.example"
  runtime       = "nodejs12.x"
}

resource "aws_kms_key" "test" {
  description             = "Terraform acc test %[1]s"
  deletion_window_in_days = 7

  policy = <<POLICY
{
  "Version": "2012-10-17",
  "Id": "kms-tf-1",
  "Statement": [
    {
      "Sid": "Enable IAM User Permissions",
      "Effect": "Allow",
      "Principal": {
        "AWS": "*"
      },
      "Action": "kms:*",
      "Resource": "*"
    }
  ]
}
POLICY
}
`, name)
}

func testAccAWSCognitoUserPoolConfig_withLambdaConfig(name string) string {
	return testAccAWSCognitoUserPoolConfigLambdaConfigBase(name) + fmt.Sprintf(`
resource "aws_cognito_user_pool" "test" {
  name = %[1]q

  lambda_config {
    create_auth_challenge          = aws_lambda_function.test.arn
    custom_message                 = aws_lambda_function.test.arn
    define_auth_challenge          = aws_lambda_function.test.arn
    post_authentication            = aws_lambda_function.test.arn
    post_confirmation              = aws_lambda_function.test.arn
    pre_authentication             = aws_lambda_function.test.arn
    pre_sign_up                    = aws_lambda_function.test.arn
    pre_token_generation           = aws_lambda_function.test.arn
    user_migration                 = aws_lambda_function.test.arn
    verify_auth_challenge_response = aws_lambda_function.test.arn
  }
}
`, name)
}

func testAccAWSCognitoUserPoolConfig_withLambdaConfigUpdated(name string) string {
	return testAccAWSCognitoUserPoolConfigLambdaConfigBase(name) + fmt.Sprintf(`
resource "aws_lambda_function" "second" {
  filename      = "test-fixtures/lambdatest.zip"
  function_name = "%[1]s_second"
  role          = aws_iam_role.test.arn
  handler       = "exports.example"
  runtime       = "nodejs12.x"
}

resource "aws_cognito_user_pool" "test" {
  name = %[1]q

  lambda_config {
    create_auth_challenge          = aws_lambda_function.second.arn
    custom_message                 = aws_lambda_function.second.arn
    define_auth_challenge          = aws_lambda_function.second.arn
    post_authentication            = aws_lambda_function.second.arn
    post_confirmation              = aws_lambda_function.second.arn
    pre_authentication             = aws_lambda_function.second.arn
    pre_sign_up                    = aws_lambda_function.second.arn
    pre_token_generation           = aws_lambda_function.second.arn
    user_migration                 = aws_lambda_function.second.arn
    verify_auth_challenge_response = aws_lambda_function.second.arn
  }
}
`, name)
}

func testAccAWSCognitoUserPoolLambdaConfigEmailSenderConfig(name string) string {
	return testAccAWSCognitoUserPoolConfigLambdaConfigBase(name) + fmt.Sprintf(`
resource "aws_cognito_user_pool" "test" {
  name = %[1]q

  lambda_config {
    kms_key_id = aws_kms_key.test.arn

    custom_email_sender {
      lambda_arn     = aws_lambda_function.test.arn
      lambda_version = "V1_0"
    }
  }
}
`, name)
}

func testAccAWSCognitoUserPoolLambdaConfigEmailSenderConfigUpdated(name string) string {
	return testAccAWSCognitoUserPoolConfigLambdaConfigBase(name) + fmt.Sprintf(`
resource "aws_lambda_function" "second" {
  filename      = "test-fixtures/lambdatest.zip"
  function_name = "%[1]s_second"
  role          = aws_iam_role.test.arn
  handler       = "exports.example"
  runtime       = "nodejs12.x"
}

resource "aws_cognito_user_pool" "test" {
  name = %[1]q

  lambda_config {
    kms_key_id = aws_kms_key.test.arn

    custom_email_sender {
      lambda_arn     = aws_lambda_function.second.arn
      lambda_version = "V1_0"
    }
  }
}
`, name)
}

func testAccAWSCognitoUserPoolLambdaConfigSMSSenderConfig(name string) string {
	return testAccAWSCognitoUserPoolConfigLambdaConfigBase(name) + fmt.Sprintf(`
resource "aws_cognito_user_pool" "test" {
  name = %[1]q

  lambda_config {
    kms_key_id = aws_kms_key.test.arn

    custom_sms_sender {
      lambda_arn     = aws_lambda_function.test.arn
      lambda_version = "V1_0"
    }
  }
}
`, name)
}

func testAccAWSCognitoUserPoolLambdaConfigSMSSenderConfigUpdated(name string) string {
	return testAccAWSCognitoUserPoolConfigLambdaConfigBase(name) + fmt.Sprintf(`
resource "aws_lambda_function" "second" {
  filename      = "test-fixtures/lambdatest.zip"
  function_name = "%[1]s_second"
  role          = aws_iam_role.test.arn
  handler       = "exports.example"
  runtime       = "nodejs12.x"
}

resource "aws_cognito_user_pool" "test" {
  name = %[1]q

  lambda_config {
    kms_key_id = aws_kms_key.test.arn

    custom_sms_sender {
      lambda_arn     = aws_lambda_function.second.arn
      lambda_version = "V1_0"
    }
  }
}
`, name)
}

func testAccAWSCognitoUserPoolConfig_withSchemaAttributes(name string) string {
	return fmt.Sprintf(`
resource "aws_cognito_user_pool" "test" {
  name = "%[1]s"

  schema {
    attribute_data_type      = "String"
    developer_only_attribute = false
    mutable                  = false
    name                     = "email"
    required                 = true

    string_attribute_constraints {
      min_length = 5
      max_length = 10
    }
  }

  schema {
    attribute_data_type      = "Boolean"
    developer_only_attribute = true
    mutable                  = false
    name                     = "mybool"
    required                 = false
  }
}
`, name)
}

func testAccAWSCognitoUserPoolConfig_withSchemaAttributesUpdated(name string, boolname string) string {
	return fmt.Sprintf(`
resource "aws_cognito_user_pool" "test" {
  name = "%[1]s"

  schema {
    attribute_data_type      = "String"
    developer_only_attribute = false
    mutable                  = false
    name                     = "email"
    required                 = true

    string_attribute_constraints {
      min_length = 5
      max_length = 10
    }
  }

  schema {
    attribute_data_type      = "Boolean"
    developer_only_attribute = true
    mutable                  = false
    name                     = %[2]q
    required                 = false
  }

  schema {
    attribute_data_type      = "Number"
    developer_only_attribute = false
    mutable                  = true
    name                     = "mynondevnumber"
    required                 = false

    number_attribute_constraints {
      min_value = 2
      max_value = 6
    }
  }
}
`, name, boolname)
}

func testAccAWSCognitoUserPoolConfig_withVerificationMessageTemplate(name string) string {
	return fmt.Sprintf(`
resource "aws_cognito_user_pool" "test" {
  name = %[1]q

  # Setting Verification template attributes like EmailMessage, EmailSubject or SmsMessage
  # will implicitly set EmailVerificationMessage, EmailVerificationSubject and SmsVerificationMessage
  # attributes.

  verification_message_template {
    default_email_option  = "CONFIRM_WITH_LINK"
    email_message         = "foo {####} bar"
    email_message_by_link = "{##foobar##}"
    email_subject         = "foobar {####}"
    email_subject_by_link = "foobar"
    sms_message           = "{####} baz"
  }
}
`, name)
}

func testAccAWSCognitoUserPoolConfig_withVerificationMessageTemplate_DefaultEmailOption(name string) string {
	return fmt.Sprintf(`
resource "aws_cognito_user_pool" "test" {
  name = %[1]q

  email_verification_message = "{####} Baz"
  email_verification_subject = "BazBaz {####}"
  sms_verification_message   = "{####} BazBazBar?"

  verification_message_template {
    default_email_option = "CONFIRM_WITH_CODE"
  }
}
`, name)
}

func testAccAWSCognitoUserPoolConfig_update(name string, mfaconfig, smsAuthMsg string) string {
	return fmt.Sprintf(`
data "aws_caller_identity" "current" {
}

resource "aws_iam_role" "test" {
  name = %[1]q
  path = "/service-role/"

  assume_role_policy = <<POLICY
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Sid": "",
      "Effect": "Allow",
      "Principal": {
        "Service": "cognito-idp.amazonaws.com"
      },
      "Action": "sts:AssumeRole",
      "Condition": {
        "StringEquals": {
          "sts:ExternalId": "${data.aws_caller_identity.current.account_id}"
        }
      }
    }
  ]
}
POLICY
}

resource "aws_iam_role_policy" "test" {
  name = %[1]q
  role = aws_iam_role.test.id

  policy = <<EOF
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": [
        "sns:publish"
      ],
      "Resource": [
        "*"
      ]
    }
  ]
}
EOF
}

resource "aws_cognito_user_pool" "test" {
  name                     = %[1]q
  auto_verified_attributes = ["email"]
  mfa_configuration        = "%[2]s"

  email_verification_message = "Foo {####} Bar"
  email_verification_subject = "FooBar {####}"
  sms_verification_message   = "{####} Baz"
  sms_authentication_message = "%[3]s"

  admin_create_user_config {
    allow_admin_create_user_only = true

    invite_message_template {
      email_message = "Your username is {username} and temporary password is {####}. "
      email_subject = "FooBar {####}"
      sms_message   = "Your username is {username} and temporary password is {####}."
    }
  }

  device_configuration {
    challenge_required_on_new_device      = true
    device_only_remembered_on_user_prompt = false
  }

  verification_message_template {
    default_email_option = "CONFIRM_WITH_CODE"
  }

  sms_configuration {
    external_id    = data.aws_caller_identity.current.account_id
    sns_caller_arn = aws_iam_role.test.arn
  }
}
`, name, mfaconfig, smsAuthMsg)
}
