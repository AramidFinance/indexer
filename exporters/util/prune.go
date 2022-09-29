package util

import (
	"context"
	"sync"
	"time"

	"github.com/algorand/indexer/idb"
	"github.com/sirupsen/logrus"
)

// Interval determines how often to delete data
type Interval int

var counter uint64

const (
	defaultTimeout uint64   = 5
	once           Interval = -1
	disabled       Interval = 0
)

// PruneConfigurations contains the configurations for data pruning
type PruneConfigurations struct {
	Rounds   uint64   `yaml:"rounds"`
	Interval Interval `yaml:"interval"`
	Timeout  uint64   `yaml:"timeout"`
}

// DataManager is a data pruning interface
type DataManager interface {
	Delete(*sync.WaitGroup, chan uint64)
}

type postgresql struct {
	config *PruneConfigurations
	db     idb.IndexerDb
	logger *logrus.Logger
	ctx    context.Context
	cf     context.CancelFunc
}

// MakeDataManager initializes resources need for removing data from data source
func MakeDataManager(ctx context.Context, cfg *PruneConfigurations, db idb.IndexerDb, logger *logrus.Logger) DataManager {
	c, cf := context.WithCancel(ctx)
	dm := postgresql{
		config: cfg,
		db:     db,
		logger: logger,
		ctx:    c,
		cf:     cf,
	}
	counter = 0

	return dm
}

// Delete removes data from the txn table in Postgres DB
func (p postgresql) Delete(wg *sync.WaitGroup, roundch chan uint64) {
	defer wg.Done()
	defer p.cf()

	timeout := defaultTimeout
	if p.config.Timeout > 0 {
		timeout = p.config.Timeout
	}
	// exec pruning job base on configured interval
	for {
		select {
		case <-p.ctx.Done():
			return
		case currentRound := <-roundch:
			p.logger.Debug("Delete: received round %d", currentRound)
			if currentRound > p.config.Rounds {
				keep := currentRound - p.config.Rounds + 1
				if (p.config.Interval == once || p.config.Interval > 0) && counter == 0 {
					// always run a delete at startup
					_, err := p.db.DeleteTransactions(p.ctx, keep, time.Duration(timeout)*time.Second)
					if err != nil {
						p.logger.Warnf("exec: data pruning err: %v", err)
						return
					}
					if p.config.Interval == once {
						return
					}
				} else if p.config.Interval > 0 && counter%uint64(p.config.Interval) == 0 {
					// then run at an interval
					_, err := p.db.DeleteTransactions(p.ctx, keep, time.Duration(timeout)*time.Second)
					if err != nil {
						p.logger.Warnf("exec: data pruning err: %v", err)
						return
					}
				} else if p.config.Interval == disabled {
					p.logger.Infof("Interval %d. Data pruning is disabled", p.config.Interval)
					return
				} else if p.config.Interval < once {
					p.logger.Infof("Interval %d. Invalid Interval value", p.config.Interval)
					return
				}
			}
			counter++
		}
	}
}
