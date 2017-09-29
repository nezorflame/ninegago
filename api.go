package ninegago

import (
	"encoding/json"
	"html"
	"strconv"
	"time"

	"github.com/pkg/errors"
	fh "github.com/valyala/fasthttp"
)

// APIClient describes 9gag client
type APIClient struct {
	AppToken   string
	DeviceUUID string
	User       *UserData
}

// NewAPIClient returns new APIClient instance
func NewAPIClient() *APIClient {
	return &APIClient{
		AppToken:   randomSHA1HexStr(),
		DeviceUUID: randomUUIDHexStr(),
	}
}

// Login authenticates to 9GAG
// Returns user data and error
func (c *APIClient) Login(username, password string) (err error) {
	var (
		resp     *loginResponse
		respBody []byte
	)

	args := map[string]string{
		"loginMethod": "9gag",
		"loginName":   username,
		"password":    getMD5HexStr(password),
		"pushToken":   randomSHA1HexStr(),
		"language":    "en_US",
	}

	if respBody, err = c.requestGET(apiURL+loginPath+urlArgsStr(args), true); err != nil {
		return
	}

	if err = json.Unmarshal(respBody, &resp); err != nil {
		err = errors.Wrap(err, "Unknown login response format")
		return
	}

	if resp.Data.UserToken == "" {
		err = errors.New("Unable to login")
		return
	}

	c.AppToken = resp.Data.UserToken
	c.User = &resp.Data.User
	return
}

// GetHotPosts retreives N top posts from chosen section
// Returns slice of posts and error
func (c *APIClient) GetHotPosts(sectionType string, count int) (posts []PostData, err error) {
	var respBody []byte
	args := map[string]string{
		"group":      "1",
		"type":       sectionType,
		"itemCount":  strconv.Itoa(count),
		"entryTypes": "animated,photo,video,album",
		"offset":     "10",
	}

	if respBody, err = c.requestGET(apiURL+postListPath+urlArgsStr(args), true); err != nil {
		return
	}

	resp := postListResponse{}
	if err = json.Unmarshal(respBody, &resp); err != nil {
		err = errors.Wrap(err, "Unknown login response format")
		return
	}

	for _, p := range resp.Data.Posts {
		p.Title = html.UnescapeString(p.Title)
		posts = append(posts, p)
	}

	return
}

func (c *APIClient) requestGET(uri string, sign bool) (result []byte, err error) {
	ts := time.Now().Unix() * 1000

	req := fh.AcquireRequest()
	req.SetRequestURI(uri)
	req.Header.SetMethod("GET")
	req.Header.Set("Connection", "keep-alive")
	req.Header.Set("Accept-Encoding", "gzip, deflate")
	req.Header.Set("Accept", "*/*")
	req.Header.Set("9GAG-9GAG_TOKEN", c.AppToken)
	req.Header.Set("9GAG-TIMESTAMP", strconv.FormatInt(ts, 10))
	req.Header.Set("9GAG-APP_ID", appID)
	req.Header.Set("X-Package-ID", appID)
	req.Header.Set("9GAG-DEVICE_UUID", c.DeviceUUID)
	req.Header.Set("X-Device-UUID", c.DeviceUUID)
	req.Header.Set("9GAG-DEVICE_TYPE", deviceType)
	req.Header.Set("9GAG-BUCKET_NAME", bucketName)

	if sign {
		req.Header.Set("9GAG-REQUEST-SIGNATURE", formReqSignature(ts, c.DeviceUUID))
	}

	defer fh.ReleaseRequest(req)

	resp := fh.AcquireResponse()
	if err = fh.Do(req, resp); err != nil {
		err = errors.Wrap(err, "Unable to post response")
		return
	}
	defer fh.ReleaseResponse(resp)

	if result, err = resp.BodyGunzip(); err != nil {
		err = errors.Wrap(err, "Unable to read response")
	}

	return
}
