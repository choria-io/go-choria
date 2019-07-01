package network

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"sync"

	"github.com/nats-io/jwt"
	gnatsd "github.com/nats-io/nats-server/v2/server"
)

// reads account JWT files from a directory, implements gnatsd.AccountResolver
type dirAccountStore struct {
	srv   accountNotificationReceiver
	store string
}

type accountNotificationReceiver interface {
	LookupAccount(name string) (*gnatsd.Account, error)
	UpdateAccountClaims(a *gnatsd.Account, ac *jwt.AccountClaims)
}

func newDirAccountStore(s accountNotificationReceiver, store string) (as *dirAccountStore, err error) {
	return &dirAccountStore{
		srv:   s,
		store: store,
	}, nil
}

func (f *dirAccountStore) Start(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()

	// TODO monitor files
	for {
		select {
		case <-ctx.Done():
			return
		}
	}
}

func (f *dirAccountStore) Stop() {
	// noop till we have file notify
}

// Fetch implements gnatsd.AccountResolver
func (f *dirAccountStore) Fetch(name string) (jwt string, err error) {
	path := filepath.Join(f.store, name) + ".jwt"
	dat, err := ioutil.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("could not retrieve account '%s' from %s: %s", name, path, err)
	}

	return string(dat), nil
}

// Store implements gnatsd.AccountResolver
func (f *dirAccountStore) Store(name string, jwt string) error {
	return errors.New("dirAccountStore does not support writing")
}
