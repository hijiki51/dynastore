// Copyright 2017 Matt Ho
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//
package dynastore

import (
	"bytes"
	"encoding/base64"
	"encoding/gob"

	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/gorilla/securecookie"
	"github.com/gorilla/sessions"
)

type codecSerializer struct {
	codecs []securecookie.Codec
}

func (c *codecSerializer) marshal(name string, session *sessions.Session) (map[string]types.AttributeValue, error) {
	values, err := securecookie.EncodeMulti(name, session.Values, c.codecs...)
	if err != nil {
		return nil, errEncodeFailed
	}

	av := map[string]types.AttributeValue{
		idField:     &types.AttributeValueMemberS{Value: session.ID},
		valuesField: &types.AttributeValueMemberS{Value: values},
	}

	if session.Options != nil {
		options, err := attributevalue.Marshal(session.Options)
		if err != nil {
			return nil, err
		}
		av[optionsField] = options
	}

	return av, nil
}

func (c *codecSerializer) unmarshal(name string, in map[string]types.AttributeValue, session *sessions.Session) error {
	if len(in) == 0 {
		return errNotFound
	}

	// id
	av, ok := in[idField]
	if !ok {
		return errMalformedSession
	}

	var id string

	err := attributevalue.Unmarshal(av, &id)
	if err != nil {
		return errMalformedSession
	}

	// payload

	av, ok = in[valuesField]
	if !ok {
		return errMalformedSession
	}

	var pl string

	err = attributevalue.Unmarshal(av, &pl)
	if err != nil {
		return errMalformedSession
	}

	values := map[interface{}]interface{}{}
	err = securecookie.DecodeMulti(name, pl, &values, c.codecs...)
	if err != nil {
		return errDecodeFailed
	}

	session.IsNew = false
	session.ID = id
	session.Values = values

	// options

	av, ok = in[optionsField]
	if ok {
		options := &sessions.Options{}
		err = attributevalue.Unmarshal(av, options)
		if err != nil {
			return err
		}
		session.Options = options
	}

	return nil
}

type gobSerializer struct {
}

func (d *gobSerializer) marshal(name string, session *sessions.Session) (map[string]types.AttributeValue, error) {
	buf := &bytes.Buffer{}
	err := gob.NewEncoder(buf).Encode(session.Values)
	if err != nil {
		return nil, errEncodeFailed
	}
	values := base64.StdEncoding.EncodeToString(buf.Bytes())

	av := map[string]types.AttributeValue{
		idField:     &types.AttributeValueMemberS{Value: session.ID},
		valuesField: &types.AttributeValueMemberS{Value: values},
	}

	// encode options

	if session.Options != nil {
		options, err := attributevalue.Marshal(session.Options)
		if err != nil {
			return nil, err
		}
		av[optionsField] = options
	}

	return av, nil
}

func (d *gobSerializer) unmarshal(name string, in map[string]types.AttributeValue, session *sessions.Session) error {
	if len(in) == 0 {
		return errNotFound
	}

	// id
	av, ok := in[idField]
	if !ok {
		return errMalformedSession
	}
	var id string

	err := attributevalue.Unmarshal(av, &id)
	if err != nil {
		return errMalformedSession
	}

	// payload

	av, ok = in[valuesField]
	if !ok {
		return errMalformedSession
	}

	var pl string

	err = attributevalue.Unmarshal(av, &pl)
	if err != nil {
		return errMalformedSession
	}

	data, err := base64.StdEncoding.DecodeString(pl)
	if err != nil {
		return errDecodeFailed
	}
	values := map[interface{}]interface{}{}
	err = gob.NewDecoder(bytes.NewReader(data)).Decode(&values)
	if err != nil {
		return errDecodeFailed
	}

	session.IsNew = false
	session.ID = id
	session.Values = values

	// options

	av, ok = in[optionsField]
	if ok {
		options := &sessions.Options{}
		err = attributevalue.Unmarshal(av, options)
		if err != nil {
			return err
		}
		session.Options = options
	}

	return nil
}
