/*
 * Copyright 2018 The Service Manager Authors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package filter

import (
	"context"
	"github.com/Peripli/service-manager/api/filters/authn/basic"
	"github.com/Peripli/service-manager/pkg/types"
	"github.com/Peripli/service-manager/pkg/util"
	"github.com/Peripli/service-manager/pkg/web"
)

type basicAuthFilter struct {
	web.Filter
}

// NewBasicAuthFilter returns a filter configured to authenticate access based on the provided username and password
func NewBasicAuthFilter(username, password string) web.Filter {
	return &basicAuthFilter{
		Filter: basic.NewFilter(&inMemoryCredentialsStorage{username: username, password: password}, &noOpEncrypter{}),
	}
}

func (ba *basicAuthFilter) FilterMatchers() []web.FilterMatcher {
	return []web.FilterMatcher{
		{
			Matchers: []web.Matcher{
				web.Path(web.OSBURL + "/**"),
			},
		},
	}
}

type noOpEncrypter struct {
}

func (*noOpEncrypter) Encrypt(ctx context.Context, plaintext []byte) ([]byte, error) {
	return plaintext, nil
}

func (*noOpEncrypter) Decrypt(ctx context.Context, ciphertext []byte) ([]byte, error) {
	return ciphertext, nil
}

type inMemoryCredentialsStorage struct {
	username, password string
}

func (p *inMemoryCredentialsStorage) Get(ctx context.Context, username string) (*types.Credentials, error) {
	if username != p.username {
		return nil, util.ErrNotFoundInStorage
	}
	return &types.Credentials{
		Basic: &types.Basic{
			Password: p.password,
			Username: p.username,
		},
	}, nil
}
