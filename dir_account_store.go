package network

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/nats-io/jwt"
	gnatsd "github.com/nats-io/nats-server/v2/server"
	nsc "github.com/nats-io/nsc/cmd/store"
)

// reads account JWT files from a NSC format directory, implements gnatsd.AccountResolver
type dirAccountStore struct {
	srv   accountNotificationReceiver
	store string
	nsc   *nsc.Store

	sync.Mutex
}

type accountNotificationReceiver interface {
	LookupAccount(name string) (*gnatsd.Account, error)
	UpdateAccountClaims(a *gnatsd.Account, ac *jwt.AccountClaims)
}

func newDirAccountStore(s accountNotificationReceiver, store string) (as *dirAccountStore, err error) {
	nscStore, err := nsc.LoadStore(store)
	if err != nil {
		return nil, fmt.Errorf("could not load NSC format store %s: %s", store, err)
	}

	return &dirAccountStore{
		srv:   s,
		store: store,
		nsc:   nscStore,
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
	f.Lock()
	defer f.Unlock()

	infos, err := f.nsc.List(nsc.Accounts)
	if err != nil {
		return "", err
	}

	for _, i := range infos {
		if i.IsDir() {
			c, err := f.nsc.LoadClaim(nsc.Accounts, i.Name(), nsc.JwtName(i.Name()))
			if err != nil {
				return "", err
			}

			if c != nil {
				if c.Subject == name {
					data, err := f.nsc.Read(nsc.Accounts, i.Name(), nsc.JwtName(i.Name()))
					if err != nil {
						return "", err
					}

					return string(data), nil
				}
			}
		}
	}

	return "", fmt.Errorf("no matching JWT found for %s", name)
}

// Store implements gnatsd.AccountResolver
func (f *dirAccountStore) Store(name string, jwt string) error {
	return errors.New("dirAccountStore does not support writing")
}
