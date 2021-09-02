package inmemory

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"sync"
	"time"

	"github.com/KonstantinGasser/ditm/examples/account_balance_example/service_a/internal/domain"
	"github.com/KonstantinGasser/ditm/examples/account_balance_example/service_a/pkg/helper"
)

const (
	// maxConcurrentAccess controls the number of paralalle
	// accesses on a given resource
	maxConcurrentAccess = 1
)

var (
	ErrNoSuchAccount     = fmt.Errorf("account not found")
	ErrDuplicatedAccount = fmt.Errorf("account with ID alredy exists")
)

type Resource struct {
	value domain.Account
	// allows to limit parallel access
	// on resource
	acquire chan struct{}
}

type DB struct {
	sync.Mutex
	accounts map[int]*Resource
}

func New() *DB {
	return &DB{
		accounts: make(map[int]*Resource),
	}
}

// Acquire looks up the requested account and if found locks its resource
func (db *DB) Acquire(ctx context.Context, ID int) (domain.Account, error) {
	resource, err := db.get(ctx, ID)
	if err != nil {
		return domain.Account{}, err
	}
	fmt.Println(db.accounts)
	// DEBUG OUTPUT
	defer log.Printf("[db:Acquire:%v] locking for ID:%d\n", ctx.Value(helper.CtxReqIP), ID)
	log.Printf("[db:Acquire:%v] waiting for lock for ID:%d\n", ctx.Value(helper.CtxReqIP), ID)
	// acquire resource / or wait until available
	select {
	case resource.acquire <- struct{}{}:
	case <-ctx.Done():
		return domain.Account{}, ctx.Err()
	}

	return resource.value, nil
}

// Realse frees a resource lock
func (db *DB) Release(ctx context.Context, ID int) error {
	resource, err := db.get(ctx, ID)
	if err != nil {
		return err
	}
	// DEBUG OUTPUT
	defer log.Printf("[db:Release:%v] unlocking for ID:%d\n", ctx.Value(helper.CtxReqIP), ID)
	// DEBUG: MAKING THINGS OBVIOUSE :)
	time.Sleep(time.Duration(rand.Intn(5)+2) * time.Second)
	<-resource.acquire

	return nil
}

func (db *DB) Create(ctx context.Context, account domain.Account) error {
	defer db.Unlock()
	db.Lock()

	_, ok := db.accounts[account.ID]
	if ok {
		return ErrDuplicatedAccount
	}
	db.accounts[account.ID] = &Resource{
		value:   account,
		acquire: make(chan struct{}, maxConcurrentAccess),
	}
	return nil
}

// Save stores an account under its ID. (Accounts will always be overwritten!)
func (db *DB) Save(ctx context.Context, account domain.Account) error {
	defer db.Unlock()
	db.Lock()

	// update account but do not create a new channel if
	// resource exisists
	cache, ok := db.accounts[account.ID]
	if ok {
		cache.value = account
		return nil
	}
	db.accounts[account.ID] = &Resource{
		value:   account,
		acquire: make(chan struct{}, maxConcurrentAccess),
	}
	return nil
}

// get retruns a resource from the DB
func (db *DB) get(ctx context.Context, uuid int) (*Resource, error) {
	// defer db.Unlock()
	// db.Lock()

	resource, ok := db.accounts[uuid]
	if !ok {
		return nil, ErrNoSuchAccount
	}
	return resource, nil
}
