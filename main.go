package main

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"os/user"
	"path"
	"path/filepath"
	"strings"

	"github.com/koding/multiconfig"
)

const certPrefix = "certificates"

// note: according to the chosen dns, corresponding envs need to be set
// run `lego dnshelp` for help on usage
type confs struct {
	// lets encrypt configs
	Email       string
	Domains     string
	DNS         string
	Renew       bool
	Path        string `default:"~/.lego"`
	SkipNewCert bool   // if certs have been fetched manually, set it to true

	// qiniu configs
	CertName    string `required:"true"`
	QiniuDomain string `required:"true"` // domain on qiniu
	ExistCertID string // if qiniu cert has been created, set it to the id to use. this means only updating domain with the cert
}

func main() {
	m := multiconfig.New()
	conf := new(confs)
	m.MustLoad(conf)

	domains := getDomains(conf.Domains)

	if strings.Contains(conf.Path, "~") {
		usr, err := user.Current()
		if err != nil {
			fmt.Println("fail to normalize path, err:", err)
			return
		}
		conf.Path = strings.Replace(conf.Path, "~", usr.HomeDir, 1)
	}

	if !conf.SkipNewCert {
		if err := generateCerts(conf.Email, conf.Path,
			conf.DNS, conf.Renew, domains); err != nil {
			fmt.Println("fail to generate certificates, err: ", err)
			return
		}
		fmt.Println("Certificates have been generated successfully!")
	}

	accessKey := os.Getenv("QINIU_ACCESS_KEY")
	secretKey := os.Getenv("QINIU_SECRET_KEY")
	client := newQiniuClient(accessKey, secretKey)

	certID := conf.ExistCertID
	// if certID is empty, then do the upload
	if certID == "" {
		var err error
		certID, err = uploadCert(client, conf.CertName, conf.Path, domains)
		if err != nil {
			fmt.Println("fail to upload certificates, err: ", err)
			return
		}
		fmt.Println("Certificates have been uploaded successfully!")
	}

	if err := updateCert(client, conf.QiniuDomain, certID); err != nil {
		fmt.Println("fail to update certificates, err: ", err)
		return
	}
	fmt.Println("Certificates have been updated successfully!")

	fmt.Println("All done!")
}

func getDomains(confDomains string) []string {
	rawDomains := strings.Split(confDomains, ",")
	domains := make([]string, 0)
	for _, d := range rawDomains {
		d = strings.Trim(d, " ")
		if d != "" {
			domains = append(domains, d)
		}
	}
	return domains
}

func generateCerts(email, path, dns string, renew bool, domains []string) error {
	path, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	args := []string{
		"--email=" + email,
		"--dns=" + dns,
		"--path=" + path,
		"--accept-tos",
		"--dns.disable-cp",
	}
	for _, d := range domains {
		args = append(args, "--domains="+d)
	}
	if renew {
		args = append(args, "renew")
	} else {
		args = append(args, "run")
	}

	cmd := exec.Command("lego", args...)
	cmd.Env = os.Environ()
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func uploadCert(client *qiniuClient, certName, certPath string,
	domains []string) (string, error) {
	if len(domains) == 0 {
		return "", errors.New("no Domains provided. Don't know which is the cert file")
	}
	commonName := domains[0]
	// change *.xxx.com to _.xxx.com
	filePrefix := strings.Replace(commonName, "*", "_", -1)
	priKeyFile := path.Join(certPath, certPrefix, filePrefix+".key")
	certFile := path.Join(certPath, certPrefix, filePrefix+".crt")

	pri, err := ioutil.ReadFile(priKeyFile)
	if err != nil {
		return "", err
	}
	ca, err := ioutil.ReadFile(certFile)
	if err != nil {
		return "", err
	}
	resp, err := client.uploadCert(certName, commonName, string(pri),
		string(ca))
	if err != nil {
		return "", err
	}

	return resp.CertID, nil
}

func updateCert(client *qiniuClient, qiniuDomain, certID string) error {
	resp, err := client.getDomainInfo(qiniuDomain)
	if err != nil {
		return err
	}

	if resp.Protocol == "http" {
		return client.domainToHTTPS(qiniuDomain, certID)
	}

	return client.domainUpdateCert(qiniuDomain, certID)
}
