package main

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha1"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/asn1"
	"fmt"
	"log"
	"math"
	"math/big"
	"net"
	"net/http"
	"time"

	"io/ioutil"

	"github.com/pkg/errors"
)

func HelloServer(w http.ResponseWriter, req *http.Request) {
	fmt.Println("test")
	w.Header().Set("Content-Type", "text/plain")
	w.Write([]byte("Hello from the dark side.\n"))
}

func main() {
	fmt.Println("start")
	http.HandleFunc("/readyz", HelloServer)
	err := http.ListenAndServeTLS(":8443", "server_org.crt", "server_org.key", nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}

const (
	keySize = 2048

	// Validity4eDay sets the validity of a cert to 96 hours.
	Validity4Day = time.Hour * 96

	// Validity4Year sets the validity of a cert to 4 year.
	Validity4Year = Validity4Day * 365

	// ValidityTenYears sets the validity of a cert to 10 years.
	ValidityTenYears = Validity4Year * 3
)

type CertCfg struct {
	DNSNames     []string
	ExtKeyUsages []x509.ExtKeyUsage
	IPAddresses  []net.IP
	KeyUsages    x509.KeyUsage
	Subject      pkix.Name
	Validity     time.Duration
	IsCA         bool
}

type rsaPublicKey struct {
	N *big.Int
	E int
}

func genCA() error {
	cfg := &CertCfg{
		Subject:   pkix.Name{CommonName: "kube-apiserver-lb-signer", OrganizationalUnit: []string{"openshift"}},
		KeyUsages: x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		Validity:  ValidityTenYears,
		IsCA:      true,
	}

	return generate(cfg, "kube-apiserver-lb-signer")
}

// certs in original use case:
// Issuer: OU = openshift, CN = kube-apiserver-lb-signer
// Subject: O = kube-master, CN = system:kube-apiserver

func generate(cfg *CertCfg, filenameBase string) error {
	key, crt, err := generateSelfSignedCertificate(cfg)
	if err != nil {
		return errors.Wrap(err, "failed to generate self-signed cert/key pair")
	}

	return writeFiles(filenameBase, key, crt)
}

func writeFiles(fileBase string, key *rsa.PrivateKey, crt *x509.Certificate) error {
	err := ioutil.WriteFile(fileBase+".key", crt.Raw, 0644)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(fileBase+".crt", crt.Raw, 0644)
}

// GenerateSelfSignedCertificate generates a key/cert pair defined by CertCfg.
func generateSelfSignedCertificate(cfg *CertCfg) (*rsa.PrivateKey, *x509.Certificate, error) {
	key, err := PrivateKey()
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to generate private key")
	}

	crt, err := SelfSignedCertificate(cfg, key)
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to create self-signed certificate")
	}
	return key, crt, nil
}

// PrivateKey generates an RSA Private key and returns the value
func PrivateKey() (*rsa.PrivateKey, error) {
	rsaKey, err := rsa.GenerateKey(rand.Reader, keySize)
	if err != nil {
		return nil, errors.Wrap(err, "error generating RSA private key")
	}

	return rsaKey, nil
}

// SelfSignedCertificate creates a self signed certificate
func SelfSignedCertificate(cfg *CertCfg, key *rsa.PrivateKey) (*x509.Certificate, error) {
	serial, err := rand.Int(rand.Reader, new(big.Int).SetInt64(math.MaxInt64))
	if err != nil {
		return nil, err
	}
	cert := x509.Certificate{
		BasicConstraintsValid: true,
		IsCA:                  cfg.IsCA,
		KeyUsage:              cfg.KeyUsages,
		NotAfter:              time.Now().Add(cfg.Validity),
		NotBefore:             time.Now().AddDate(0, 0, -2),
		SerialNumber:          serial,
		Subject:               cfg.Subject,
	}
	// verifies that the CN and/or OU for the cert is set
	if len(cfg.Subject.CommonName) == 0 || len(cfg.Subject.OrganizationalUnit) == 0 {
		return nil, errors.Errorf("certification's subject is not set, or invalid")
	}
	pub := key.Public()
	cert.SubjectKeyId, err = generateSubjectKeyID(pub)
	if err != nil {
		return nil, errors.Wrap(err, "failed to set subject key identifier")
	}
	certBytes, err := x509.CreateCertificate(rand.Reader, &cert, &cert, key.Public(), key)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create certificate")
	}
	return x509.ParseCertificate(certBytes)
}

// generateSubjectKeyID generates a SHA-1 hash of the subject public key.
func generateSubjectKeyID(pub crypto.PublicKey) ([]byte, error) {
	var publicKeyBytes []byte
	var err error

	switch pub := pub.(type) {
	case *rsa.PublicKey:
		publicKeyBytes, err = asn1.Marshal(rsaPublicKey{N: pub.N, E: pub.E})
		if err != nil {
			return nil, errors.Wrap(err, "failed to Marshal ans1 public key")
		}
	case *ecdsa.PublicKey:
		publicKeyBytes = elliptic.Marshal(pub.Curve, pub.X, pub.Y)
	default:
		return nil, errors.New("only RSA and ECDSA public keys supported")
	}

	hash := sha1.Sum(publicKeyBytes)
	return hash[:], nil
}
