package main

import (
	"fmt"
	"reflect"

	"github.com/aws/aws-lambda-go/cfn"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/iam"
)

func (p *identityProvider) createOpenIDConnectProvider(event cfn.Event) (physicalResourceID string, data map[string]interface{}, err error) {

	data = map[string]interface{}{"ProviderArn": p.providerARN}

	req := &iam.CreateOpenIDConnectProviderInput{
		Url:            &p.issuerURL,
		ClientIDList:   p.clientIDList,
		ThumbprintList: p.thumbprintList,
	}

	// handle already exist provider to implement idempotent function
	_, err = p.client.CreateOpenIDConnectProvider(req)
	if err != nil {
		if awsErr, ok := err.(awserr.Error); ok {
			switch awsErr.Code() {
			case iam.ErrCodeEntityAlreadyExistsException:
				return p.providerName, data, nil
			default:
				return event.PhysicalResourceID, map[string]interface{}{}, fmt.Errorf("error creating OpenIDConnect provider (%v): %v", req, err)
			}
		}
	}

	return p.providerName, data, nil
}

func (p *identityProvider) updateOpenIDConnectProvider(event cfn.Event) (physicalResourceID string, data map[string]interface{}, err error) {

	data = map[string]interface{}{"ProviderArn": p.providerARN}

	// parse old client IDs and Thumbprints
	var oldClientIDList []*string
	var oldThumbprintList []*string

	if val, ok := event.OldResourceProperties["ClientIDList"]; ok {
		for _, c := range val.([]interface{}) {
			cp := c.(string)
			if cp != "" {
				oldClientIDList = append(oldClientIDList, &cp)
			}
		}
	}

	if val, ok := event.OldResourceProperties["ThumbprintList"]; ok {
		for _, t := range val.([]interface{}) {
			tp := t.(string)
			if tp != "" {
				oldThumbprintList = append(oldThumbprintList, &tp)
			}
		}
	}

	// updating OIDC is not straightforward, can't updated in batch
	// update Thumbprints if changed
	if !reflect.DeepEqual(oldThumbprintList, p.thumbprintList) {
		req := &iam.UpdateOpenIDConnectProviderThumbprintInput{
			OpenIDConnectProviderArn: &p.providerARN,
			ThumbprintList:           p.thumbprintList,
		}
		_, err = p.client.UpdateOpenIDConnectProviderThumbprint(req)
		if err != nil {
			return event.PhysicalResourceID, map[string]interface{}{}, fmt.Errorf("error updating OpenIDConnect provider (%v): %v", req, err)
		}
	}

	// update IDs if changed
	// AWS allows only either append or remove of client ID to current IDs
	// so we remove the old IDs and then add the new IDs
	if !reflect.DeepEqual(oldClientIDList, p.clientIDList) {
		// remove old IDs
		for _, c := range oldClientIDList {
			req := &iam.RemoveClientIDFromOpenIDConnectProviderInput{
				OpenIDConnectProviderArn: &p.providerARN,
				ClientID:                 c,
			}
			_, err = p.client.RemoveClientIDFromOpenIDConnectProvider(req)
			if err != nil {
				return event.PhysicalResourceID, data, fmt.Errorf("error updating OpenIDConnect provider (%v): %v", req, err)
			}
		}
		// add new IDs
		for _, c := range p.clientIDList {
			req := &iam.AddClientIDToOpenIDConnectProviderInput{
				OpenIDConnectProviderArn: &p.providerARN,
				ClientID:                 c,
			}
			_, err = p.client.AddClientIDToOpenIDConnectProvider(req)
			if err != nil {
				return event.PhysicalResourceID, data, fmt.Errorf("error updating OpenIDConnect provider (%v): %v", req, err)
			}
		}
	}

	return event.PhysicalResourceID, data, nil
}

func (p *identityProvider) deleteOpenIDConnectProvider(event cfn.Event) (physicalResourceID string, data map[string]interface{}, err error) {

	data = map[string]interface{}{"ProviderArn": p.providerARN}

	req := &iam.DeleteOpenIDConnectProviderInput{
		OpenIDConnectProviderArn: &p.providerARN,
	}

	_, err = p.client.DeleteOpenIDConnectProvider(req)
	if err != nil {
		return event.PhysicalResourceID, map[string]interface{}{}, fmt.Errorf("error deleting OpenIDConnect provider (%v): %v", req, err)
	}

	return event.PhysicalResourceID, data, nil
}
