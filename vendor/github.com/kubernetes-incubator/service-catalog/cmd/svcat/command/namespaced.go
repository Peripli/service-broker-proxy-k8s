/*
Copyright 2018 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package command

// Namespaced is the base command of all svcat commands that are namespace scoped.
type Namespaced struct {
	*Context
	Namespace string
}

// NewNamespacedCommand from context.
func NewNamespacedCommand(cxt *Context) *Namespaced {
	return &Namespaced{Context: cxt}
}

// GetContext retrieves the command's context.
func (c *Namespaced) GetContext() *Context {
	return c.Context
}

// SetNamespace sets the effective namespace for the command.
func (c *Namespaced) SetNamespace(namespace string) {
	c.Namespace = namespace
}
