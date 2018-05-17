package certificateGenerator

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	b64 "encoding/base64"
	"encoding/pem"
	"fmt"
	"io/ioutil"
	"log"
	"math/big"
	"net"
	"os"
	"time"
)

func GenCert(CAFilePath string, commonName string, ips []string, hostNames []string) []byte {

	fmt.Println("[GenCert] ", commonName)

	rootCertPem, err := ioutil.ReadFile(CAFilePath)
	if err != nil {
		log.Fatalf("failed to read root cert: %s", err)
	}

	rootCertBytes, rest := pem.Decode(rootCertPem)

	rootCert, err := x509.ParseCertificate(rootCertBytes.Bytes)
	if err != nil {
		log.Fatalf("failed to parse root cert: %s", err)
	}

	rootPrivateKeyBytes, _ := pem.Decode(rest)
	rootPrivateKey, err := x509.ParsePKCS1PrivateKey(rootPrivateKeyBytes.Bytes)
	if err != nil {
		log.Fatalf("failed to parse root cert: %s", err)
	}

	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		log.Fatalf("failed to generate private key: %s", err)
	}

	notBefore := time.Now()
	notAfter := notBefore.Add(365 * 24 * time.Hour)
	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, snErr := rand.Int(rand.Reader, serialNumberLimit)
	if snErr != nil {
		log.Fatalf("failed to generate serial number: %s", err)
	}

	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			CommonName:   commonName,
			Organization: []string{"University of Nottingham"},
			Country:      []string{"UK"},
		},
		NotBefore: notBefore,
		NotAfter:  notAfter,

		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		AuthorityKeyId:        rootCert.AuthorityKeyId,
		RawIssuer:             rootCert.RawIssuer,
	}
	for _, ip := range ips {
		template.IPAddresses = append(template.IPAddresses, net.ParseIP(ip))
	}
	for _, h := range hostNames {
		template.DNSNames = append(template.DNSNames, h)
	}

	template.IsCA = false

	derBytes, derErr := x509.CreateCertificate(rand.Reader, &template, rootCert, &priv.PublicKey, rootPrivateKey)
	if derErr != nil {
		log.Fatalf("Failed to create certificate: %s", derErr)
	}

	cert := new(bytes.Buffer)
	pem.Encode(cert, &pem.Block{Type: "CERTIFICATE", Bytes: derBytes})

	b := x509.MarshalPKCS1PrivateKey(priv)
	pem.Encode(cert, &pem.Block{Type: "RSA PRIVATE KEY", Bytes: b})

	asn1Bytes, pubErr := x509.MarshalPKIXPublicKey(&priv.PublicKey)
	if pubErr != nil {
		log.Fatalf("gen public key %s", pubErr)
	}
	pem.Encode(cert, &pem.Block{Type: "PUBLIC KEY", Bytes: asn1Bytes})

	return cert.Bytes()
}

func GenCertToFile(CAFilePath string, commonName string, ips []string, hostNames []string, outputFilePath string) {

	cert := GenCert(CAFilePath, commonName, ips, hostNames)

	certOut, err := os.Create(outputFilePath)
	if err != nil {
		log.Fatalf("failed to open "+outputFilePath+" for writing: %s", err)
	}
	_, err = certOut.Write(cert)
	if err != nil {
		log.Fatalf("Error writing to file  "+outputFilePath+" %s", err)
	}
	certOut.Close()

}

func GenRootCA(CAFilePath string) {

	priv, err := rsa.GenerateKey(rand.Reader, 2048)

	if err != nil {
		log.Fatalf("failed to generate private key: %s", err)
	}
	notBefore := time.Now()
	notAfter := notBefore.Add(365 * 24 * time.Hour)
	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, snErr := rand.Int(rand.Reader, serialNumberLimit)
	if snErr != nil {
		log.Fatalf("failed to generate serial number: %s", err)
	}

	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			CommonName:   "Databox",
			Organization: []string{"University of Nottingham"},
			Country:      []string{"UK"},
		},
		NotBefore: notBefore,
		NotAfter:  notAfter,

		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}

	template.IsCA = true
	template.KeyUsage |= x509.KeyUsageCertSign

	derBytes, derErr := x509.CreateCertificate(rand.Reader, &template, &template, &priv.PublicKey, priv)
	if derErr != nil {
		log.Fatalf("Failed to create certificate: %s", derErr)
	}

	certOut, err := os.Create(CAFilePath)
	if err != nil {
		log.Fatalf("failed to open "+CAFilePath+" for writing: %s", err)
	}

	pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: derBytes})
	b := x509.MarshalPKCS1PrivateKey(priv)
	pem.Encode(certOut, &pem.Block{Type: "RSA PRIVATE KEY", Bytes: b})
	certOut.Close()
}

func GenerateArbiterToken() []byte {
	len := 32
	data := make([]byte, len)
	_, err := rand.Read(data)
	if err != nil {
		fmt.Println("error:", err)
		return []byte{}
	}

	return data

}

func GenerateArbiterTokenToFile(outputFilePath string) {
	out, err := os.Create(outputFilePath)
	if err != nil {
		log.Fatalf("failed to open "+outputFilePath+" for writing: %s", err)
	}
	data := GenerateArbiterToken()
	out.WriteString(b64.StdEncoding.EncodeToString(data))
	out.Close()
}
