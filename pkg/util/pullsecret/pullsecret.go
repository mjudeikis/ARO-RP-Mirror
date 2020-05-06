package pullsecret

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"encoding/base64"
	"encoding/json"
	"strings"

	"github.com/Azure/ARO-RP/pkg/api"
)

type pullSecret struct {
	Auths map[string]map[string]interface{} `json:"auths,omitempty"`
}

func ParseRegistryProfiles(rps []*api.RegistryProfile) (string, error) {
	var ps pullSecret
	ps.Auths = map[string]map[string]interface{}{}

	for _, rp := range rps {
		ps.Auths[rp.Name] = map[string]interface{}{
			"auth": base64.StdEncoding.EncodeToString([]byte(rp.Username + ":" + string(rp.Password))),
		}
	}

	b, err := json.Marshal(ps)
	return string(b), err
}

// Merge returns _ps over _base.  If both _ps and _base have a given key, the
// version of it in _ps wins.
func Merge(_base, _ps string) (string, error) {
	if _base == "" {
		_base = "{}"
	}

	if _ps == "" {
		_ps = "{}"
	}

	var base, ps *pullSecret

	err := json.Unmarshal([]byte(_base), &base)
	if err != nil {
		return "", err
	}

	err = json.Unmarshal([]byte(_ps), &ps)
	if err != nil {
		return "", err
	}

	for k, v := range ps.Auths {
		if base.Auths == nil {
			base.Auths = map[string]map[string]interface{}{}
		}

		base.Auths[k] = v
	}

	b, err := json.Marshal(base)
	return string(b), err
}

func RemoveKey(_ps, key string) (string, error) {
	if _ps == "" {
		_ps = "{}"
	}

	var ps *pullSecret

	err := json.Unmarshal([]byte(_ps), &ps)
	if err != nil {
		return "", err
	}

	delete(ps.Auths, key)

	b, err := json.Marshal(ps)
	return string(b), err
}

func Validate(_ps string) error {
	if _ps == "" {
		_ps = "{}"
	}

	var ps *pullSecret

	return json.Unmarshal([]byte(_ps), &ps)
}

func GetRegistryProfiles(_ps string) ([]*api.RegistryProfile, error) {
	var ps *pullSecret
	var rps []*api.RegistryProfile

	err := json.Unmarshal([]byte(_ps), &ps)
	if err != nil {
		return nil, err
	}

	for name, auth := range ps.Auths {
		authString, err := base64.StdEncoding.DecodeString(auth["auth"].(string))
		if err != nil {
			return nil, err
		}

		rp := api.RegistryProfile{
			Name:     name,
			Username: strings.Split(string(authString), ":")[0],
			Password: api.SecureString(strings.Split(string(authString), ":")[1]),
		}
		rps = append(rps, &rp)
	}

	return rps, nil
}
