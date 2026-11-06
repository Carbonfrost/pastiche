// Copyright 2026 The Pastiche Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package model

import (
	"errors"
	"fmt"
	"io/fs"
	"strings"
	"time"

	"github.com/Carbonfrost/joe-cli/extensions/expr/expander"
	"github.com/Carbonfrost/pastiche/pkg/internal/log"
)

// ErrLoginExpired is returned for a login whose expiration time has passed
var ErrLoginExpired = errors.New("login expired")

//go:generate go tool counterfeiter -generate
//counterfeiter:generate -o ../internal/modelfakes . LoginProvider

// LoginProvider provides the logins of previously authenticated users
type LoginProvider interface {
	// GetLogin obtains the login with the given name for the given server.
	// When the login has expired, both the login and ErrLoginExpired are
	// returned.  When there is no such login, the error wraps fs.ErrNotExist.
	GetLogin(name, server string) (*Login, error)

	// ListLogins obtains all logins, including those which have expired,
	// sorted by name and server
	ListLogins() ([]*Login, error)

	// SaveLogin stores the login, replacing any login with the same name
	// and server
	SaveLogin(login *Login) error

	// ClearLogin removes the login with the given name for the given server
	ClearLogin(name, server string) error

	// ClearAllLogins removes all logins
	ClearAllLogins() error

	// GC provides garbage collection on the login provider, which is used to periodically
	// delete expired logins.
	GC() error
}

// Login provides a token for a previously authenticated user
type Login struct {
	Name   string
	Server string

	Token   string
	Expires *time.Time
}

// Expired determines whether the login has expired as of the given time
func (l *Login) Expired(now time.Time) bool {
	return l.Expires != nil && !now.Before(*l.Expires)
}

// WithLogins makes logins available to the "login" expander, which resolves
// login.<name> to the token and login.<name>.expires to its expiration time.
func WithLogins(p LoginProvider) RequestOption {
	if p == nil {
		return nil
	}
	return requestOption(func(r *requestBuilder) {
		r.logins = p
	})
}

func newLoginExpander(p LoginProvider, server string) Expander {
	if p == nil {
		return expander.Nil
	}
	return expander.Func(func(k string) any {
		// Names cannot contain dots, so the first dot separates the name
		// from the property of the login
		name, prop, hasProp := strings.Cut(k, ".")

		login, err := p.GetLogin(name, server)
		if errors.Is(err, fs.ErrNotExist) {
			return nil
		}

		if !hasProp {
			if err != nil || login == nil {
				log.Warnf("warning: unable to load login %q: %v", name, err)
				return fmt.Sprintf("<missing login %s>", name)
			}
			return login.Token
		}

		// The expiration time remains available for an expired login
		prop, timeProp, hasTimeProp := strings.Cut(prop, ".")
		if prop != "expires" || login == nil || login.Expires == nil {
			return nil
		}
		if !hasTimeProp {
			return *login.Expires
		}
		return expander.Time(*login.Expires).Expand(timeProp)
	})
}
