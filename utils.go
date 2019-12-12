package main

import (
	"crypto/sha1"
	"crypto/tls"
	"fmt"
	"net/url"
	"strings"

	"context"

	"github.com/pkg/errors"

	"github.com/aws/aws-lambda-go/lambdacontext"
)

func (p *identityProvider) setAccountID(ctx context.Context) error {

	lc, _ := lambdacontext.FromContext(ctx)
	if lc == nil {
		return fmt.Errorf("could not get account ID: nil LambdaContext; ")
	}

	if lc.InvokedFunctionArn == "" {
		return fmt.Errorf("could not get account ID: empty InvokedFunctionArn")
	}

	p.accountID = strings.Split(lc.InvokedFunctionArn, ":")[4]

	return nil
}

func getIssuerCAThumbprint(issuer string) (*string, error) {

	config := &tls.Config{InsecureSkipVerify: true}

	issuerURL, err := url.Parse(issuer)
	if err != nil {
		return nil, fmt.Errorf("unable to parse OIDC issuer's url")
	}

	if issuerURL.Scheme != "https" {
		return nil, fmt.Errorf("unsupported URL scheme %q", issuerURL.Scheme)
	}

	if issuerURL.Port() == "" {
		issuerURL.Host += ":443"
	}

	conn, err := tls.Dial("tcp", issuerURL.Host, config)
	if err != nil {
		return nil, errors.Wrapf(err, "connecting to issuer OIDC (%s)", issuerURL)
	}
	defer conn.Close()

	cs := conn.ConnectionState()
	if numCerts := len(cs.PeerCertificates); numCerts >= 1 {
		root := cs.PeerCertificates[numCerts-1]
		issuerCAThumbprint := fmt.Sprintf("%x", sha1.Sum(root.Raw))
		return &issuerCAThumbprint, nil
	}
	return nil, fmt.Errorf("unable to get OIDC issuer's certificate")
}
