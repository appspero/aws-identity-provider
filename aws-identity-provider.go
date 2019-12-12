package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/aws/aws-lambda-go/cfn"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/iam"
)

type identityProvider struct {
	client    *iam.IAM
	accountID string

	// issuerURL must begin with https
	issuerURL      string
	clientIDList   []*string
	thumbprintList []*string

	samlMetadataDocument string

	// providerName is issuerURL without https or SAML provider name
	// used as physicalResourceID
	providerName string
	providerARN  string
}

func main() {
	lambda.Start(cfn.LambdaWrap(handler))
}

// lambda handler
func handler(ctx context.Context, event cfn.Event) (physicalResourceID string, data map[string]interface{}, err error) {

	p := identityProvider{}

	// set accountID; used for provider ARN
	if err := p.setAccountID(ctx); err != nil {
		return event.PhysicalResourceID, map[string]interface{}{}, fmt.Errorf("could not get account ID: %v", err)
	}

	var providerType string

	// check ProviderType
	if val, ok := event.ResourceProperties["ProviderType"]; ok {
		providerType = val.(string)
	} else {
		return event.PhysicalResourceID, map[string]interface{}{}, fmt.Errorf("missing provider type")
	}

	// parse provider based on type
	if providerType == "OIDC" {
		err = p.parseOIDC(event)
	} else if providerType == "SAML" {
		err = p.parseSAML(event)
	} else {
		err = fmt.Errorf("invalid identity provider type: %s", providerType)
	}

	if err != nil {
		return event.PhysicalResourceID, map[string]interface{}{}, err
	}

	// AWS IAM client (uses lambda execution role)
	sess, err := session.NewSession()
	if err != nil {
		return p.providerName, map[string]interface{}{}, fmt.Errorf("Error creating AWS session: %v", err)
	}
	p.client = iam.New(sess)

	// create OIDC provider
	if event.RequestType == "Create" && providerType == "OIDC" {
		return p.createOpenIDConnectProvider(event)
	}

	// delete OIDC provider
	if event.RequestType == "Delete" && providerType == "OIDC" {
		return p.deleteOpenIDConnectProvider(event)
	}

	// update provider
	if event.RequestType == "Update" && providerType == "OIDC" {
		// update form SAML to OIDC
		if providerType != event.OldResourceProperties["ProviderType"] {
			return p.createOpenIDConnectProvider(event)
		}
		// could not update IssuerURL; so recreate the provider
		if p.issuerURL != event.OldResourceProperties["IssuerURL"] {
			return p.createOpenIDConnectProvider(event)
		}
		// update OIDC
		return p.updateOpenIDConnectProvider(event)
	}

	// create SAML provider
	if event.RequestType == "Create" && providerType == "SAML" {
		return p.createSAMLProvider(event)
	}

	// create SAML provider
	if event.RequestType == "Delete" && providerType == "SAML" {
		return p.deleteSAMLProvider(event)
	}

	// update provider
	if event.RequestType == "Update" && providerType == "SAML" {
		// update form OIDC to SAML
		if providerType != event.OldResourceProperties["ProviderType"] {
			return p.createSAMLProvider(event)
		}
		// could not update SAMLProviderName; so recreate the provider
		if p.providerName != event.OldResourceProperties["SAMLProviderName"] {
			return p.createSAMLProvider(event)
		}
		// update SAML
		return p.updateSAMLProvider(event)
	}

	return "", map[string]interface{}{}, nil
}

// parse OIDC properties
func (p *identityProvider) parseOIDC(event cfn.Event) error {

	if val, ok := event.ResourceProperties["IssuerURL"]; ok {
		p.issuerURL = val.(string)
	} else {
		return fmt.Errorf("missing OIDC IssuerURL")
	}

	// IssuerURL should start with https
	if !strings.HasPrefix(p.issuerURL, "https://") {
		return fmt.Errorf("issuer URL should start with 'https': %s", p.issuerURL)
	}

	// providerName used as physicalResourceID for OIDC provider
	p.providerName = strings.TrimPrefix(p.issuerURL, "https://")

	// providerARN used for returned data of the stack
	p.providerARN = fmt.Sprintf("arn:aws:iam::%s:oidc-provider/%s", p.accountID, p.providerName)

	// parse ClientIDList
	if val, ok := event.ResourceProperties["ClientIDList"]; ok {
		for _, c := range val.([]interface{}) {
			cp := c.(string)
			if cp != "" {
				p.clientIDList = append(p.clientIDList, &cp)
			}
		}
	} else {
		return fmt.Errorf("missing OIDC ClientIDList: at least one client ID should be specified for OIDC")
	}

	if len(p.clientIDList) == 0 {
		return fmt.Errorf("missing OIDC ClientIDList: at least one client ID should be specified for OIDC")
	}

	// parse ThumbprintList
	if val, ok := event.ResourceProperties["ThumbprintList"]; ok {
		for _, t := range val.([]interface{}) {
			tp := t.(string)
			if tp != "" {
				p.thumbprintList = append(p.thumbprintList, &tp)
			}
		}
	} else {
		// try to retrive thumbprint
		thumbprint, err := getIssuerCAThumbprint(p.issuerURL)
		if err != nil {
			return err
		}
		p.thumbprintList = []*string{thumbprint}
	}

	// try to retrive thumbprint
	if len(p.thumbprintList) == 0 {
		thumbprint, err := getIssuerCAThumbprint(p.issuerURL)
		if err != nil {
			return err
		}
		p.thumbprintList = append(p.thumbprintList, thumbprint)
	}

	return nil
}

// parse SAML properties
func (p *identityProvider) parseSAML(event cfn.Event) error {

	// providerName used as physicalResourceID of SAML provider
	if val, ok := event.ResourceProperties["SAMLProviderName"]; ok {
		p.providerName = val.(string)
	} else {
		return fmt.Errorf("missing SAMLProviderName")
	}

	if p.providerName == "" {
		return fmt.Errorf("invalid SAMLProviderName")
	}

	// providerARN used for returned data of the stack
	p.providerARN = fmt.Sprintf("arn:aws:iam::%s:saml-provider/%s", p.accountID, p.providerName)

	// parse SAML Metadata Document
	if val, ok := event.ResourceProperties["SAMLMetadataDocument"]; ok {
		p.samlMetadataDocument = val.(string)
	} else {
		return fmt.Errorf("missing SamlMetadataDocument")
	}

	if p.samlMetadataDocument == "" {
		return fmt.Errorf("invalid SamlMetadataDocument")
	}

	return nil
}
