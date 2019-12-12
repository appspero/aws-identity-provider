package main

import (
	"fmt"

	"github.com/aws/aws-lambda-go/cfn"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/iam"
)

func (p *identityProvider) createSAMLProvider(event cfn.Event) (physicalResourceID string, data map[string]interface{}, err error) {

	data = map[string]interface{}{"ProviderArn": p.providerARN}

	req := &iam.CreateSAMLProviderInput{
		Name:                 &p.providerName,
		SAMLMetadataDocument: &p.samlMetadataDocument,
	}

	// handle already exist provider to implement idempotent function
	_, err = p.client.CreateSAMLProvider(req)
	if err != nil {
		if awsErr, ok := err.(awserr.Error); ok {
			switch awsErr.Code() {
			case iam.ErrCodeEntityAlreadyExistsException:
				return p.providerName, data, nil
			default:
				return event.PhysicalResourceID, map[string]interface{}{}, fmt.Errorf("error creating SAML provider: %v", err)
			}
		}

	}

	return p.providerName, data, nil
}

func (p *identityProvider) updateSAMLProvider(event cfn.Event) (physicalResourceID string, data map[string]interface{}, err error) {

	data = map[string]interface{}{"ProviderArn": p.providerARN}

	req := &iam.UpdateSAMLProviderInput{
		SAMLProviderArn:      &p.providerARN,
		SAMLMetadataDocument: &p.samlMetadataDocument,
	}

	_, err = p.client.UpdateSAMLProvider(req)
	if err != nil {
		return event.PhysicalResourceID, map[string]interface{}{}, fmt.Errorf("error updating SAML provider: %v", err)
	}

	return event.PhysicalResourceID, data, nil
}

func (p *identityProvider) deleteSAMLProvider(event cfn.Event) (physicalResourceID string, data map[string]interface{}, err error) {

	data = map[string]interface{}{"ProviderArn": p.providerARN}

	req := &iam.DeleteSAMLProviderInput{
		SAMLProviderArn: &p.providerARN,
	}

	_, err = p.client.DeleteSAMLProvider(req)
	if err != nil {
		return event.PhysicalResourceID, map[string]interface{}{}, fmt.Errorf("error deleting SAML provider: %v", err)
	}

	return event.PhysicalResourceID, data, nil
}
