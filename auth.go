package steam_go

import (
	"errors"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"regexp"
	"strings"
)

var (
	steamLogin = "https://steamcommunity.com/openid/login"

	openMode       = "checkid_setup"
	openNS         = "http://specs.openid.net/auth/2.0"
	openIdentifier = "http://specs.openid.net/auth/2.0/identifier_select"

	validationRegexp       = regexp.MustCompile("^(http|https)://steamcommunity.com/openid/id/[0-9]{15,25}$")
	digitsExtractionRegexp = regexp.MustCompile("\\D+")
)

// OpenID ...
type OpenID struct {
	root      string
	returnUrl string
	data      url.Values
}

// NewOpenID ...
func NewOpenID(r *http.Request) *OpenID {
	id := new(OpenID)

	proto := "http://"
	if r.Header.Get("X-Forwarded-Proto") == "https" || r.TLS != nil {
		proto = "https://"
	}

	if r.Header.Get("X-Forwarded-Host") != "" {
		id.root = proto + r.Host
	} else {
		id.root = proto + r.Header.Get("X-Forwarded-Host")
	}

	uri := r.RequestURI
	if i := strings.Index(uri, "openid"); i != -1 {
		uri = uri[0 : i-1]
	}
	id.returnUrl = id.root + uri

	switch r.Method {
	case "POST":
		id.data = r.Form
	case "GET":
		id.data = r.URL.Query()
	}

	return id
}

func (id OpenID) AuthUrl() string {
	data := map[string]string{
		"openid.claimed_id": openIdentifier,
		"openid.identity":   openIdentifier,
		"openid.mode":       openMode,
		"openid.ns":         openNS,
		"openid.realm":      id.root,
		"openid.return_to":  id.returnUrl,
	}

	i := 0
	url := steamLogin + "?"
	for key, value := range data {
		url += key + "=" + value
		if i != len(data)-1 {
			url += "&"
		}
		i++
	}
	return url
}

// ValidateAndGetID ...
func (id *OpenID) ValidateAndGetID() (string, error) {
	if id.Mode() != "id_res" {
		return "", errors.New("Mode must equal to \"id_res\".")
	}

	log.Println(id.data.Get("openid.return_to"))
	log.Println(id.returnUrl)
	if id.data.Get("openid.return_to") != id.returnUrl {
		return "", errors.New("The \"return_to url\" must match the url of current request")
	}

	params := make(url.Values)
	params.Set("openid.assoc_handle", id.data.Get("openid.assoc_handle"))
	params.Set("openid.signed", id.data.Get("openid.signed"))
	params.Set("openid.sig", id.data.Get("openid.sig"))
	params.Set("openid.ns", id.data.Get("openid.ns"))

	split := strings.Split(id.data.Get("openid.signed"), ",")
	for _, item := range split {
		params.Set("openid."+item, id.data.Get("openid."+item))
	}
	params.Set("openid.mode", "check_authentication")

	resp, err := http.PostForm(steamLogin, params)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	content, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	response := strings.Split(string(content), "\n")
	if response[0] != "ns:"+openNS {
		return "", errors.New("Wrong ns in the response.")
	}
	if strings.HasSuffix(response[1], "false") {
		return "", errors.New("Unable validate openId.")
	}

	openIdUrl := id.data.Get("openid.claimed_id")
	if !validationRegexp.MatchString(openIdUrl) {
		return "", errors.New("Invalid steam id pattern.")
	}

	return digitsExtractionRegexp.ReplaceAllString(openIdUrl, ""), nil
}

// ValidateAndGetUser ...
func (id OpenID) ValidateAndGetUser(apiKey string) (*PlayerSummaries, error) {
	steamID, err := id.ValidateAndGetID()
	if err != nil {
		return nil, err
	}
	return GetPlayerSummaries(steamID, apiKey)
}

// Mode ...
func (id OpenID) Mode() string {
	return id.data.Get("openid.mode")
}
