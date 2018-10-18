package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"

	"github.com/qiniu/api.v7/auth/qbox"
)

const qiniuAPIHost = "http://api.qiniu.com"

var errQiniuDefault = errors.New("qiniu api error")

type qiniuClient struct {
	*qbox.Mac
}

type apiError struct {
	Code     int    `json:"code"`
	ErrorMsg string `json:"error"`
}

type certUploadReq struct {
	Name       string `json:"name"`
	CommonName string `json:"common_name"`
	CA         string `json:"ca"`
	Pri        string `json:"pri"`
}

type certUploadResp struct {
	CertID string `json:"certID"`
}

type domainInfoResp struct {
	Protocol string `json:"protocol"`
}

type domainCertReq struct {
	CertID     string `json:"certid"`
	ForceHTTPS bool   `json:"forceHttps"`
}

func (a apiError) Error() string {
	return fmt.Sprintf("code:%d,errmsg:%s", a.Code, a.ErrorMsg)
}

func newQiniuClient(accessKey, secretKey string) *qiniuClient {
	return &qiniuClient{Mac: qbox.NewMac(accessKey, secretKey)}
}

func (c *qiniuClient) request(method string, path string, body, respBody interface{}) error {
	urlStr := fmt.Sprintf("%s%s", qiniuAPIHost, path)

	var bodyReader io.Reader
	if body != nil {
		reqData, err := json.Marshal(body)
		if err != nil {
			return err
		}
		bodyReader = bytes.NewReader(reqData)
	}

	req, err := http.NewRequest(method, urlStr, bodyReader)
	if err != nil {
		return err
	}

	token, err := c.SignRequest(req)
	if err != nil {
		return err
	}

	req.Header.Add("Authorization", "QBox "+token)
	if body != nil {
		req.Header.Add("Content-Type", "application/json")
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var resData []byte
	if resp.StatusCode >= 400 {
		resData, err = ioutil.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("fail to read response body, resp status:%d",
				resp.StatusCode)
		}
		e := &apiError{}
		if jErr := json.Unmarshal(resData, e); jErr != nil {
			return jErr
		}
		return e
	}

	// no need to decode body
	if respBody == nil {
		return nil
	}

	resData, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	return json.Unmarshal(resData, respBody)
}

func (c *qiniuClient) uploadCert(name, domains, pri, ca string) (*certUploadResp, error) {
	cert := certUploadReq{
		Name:       name,
		CommonName: domains,
		Pri:        pri,
		CA:         ca,
	}
	resp := &certUploadResp{}
	err := c.request(http.MethodPost, "/sslcert", cert, resp)
	return resp, err
}

func (c *qiniuClient) getDomainInfo(qiniuDomainName string) (*domainInfoResp, error) {
	resp := &domainInfoResp{}
	err := c.request(http.MethodGet, "/domain/"+qiniuDomainName, nil, resp)
	return resp, err
}

func (c *qiniuClient) domainToHTTPS(qiniuDomainName, certID string) error {
	req := domainCertReq{CertID: certID}
	return c.request(http.MethodPut, "/domain/"+qiniuDomainName+"/sslize", req, nil)
}

func (c *qiniuClient) domainUpdateCert(qiniuDomainName, certID string) error {
	req := domainCertReq{CertID: certID}
	return c.request(http.MethodGet, "/domain/"+qiniuDomainName+"/httpsconf", req, nil)
}
